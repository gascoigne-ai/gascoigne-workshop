package proto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// grpcEndText, sent to a client- or bidirectional-streaming session, closes
// this side of the stream (the server may still answer). It is not valid
// JSON, so it cannot be mistaken for a message.
const grpcEndText = "end"

// Connect implements Streamer: it opens a stream for a server-, client- or
// bidirectionally streaming method. The Body is the first message (blank
// is an empty one for a server stream, none for the other kinds); a server
// stream is then closed on this side, the others take further messages
// through Session.Send. A unary method is refused.
func (GRPC) Connect(ctx context.Context, req model.Request) (Session, error) {
	conn, err := dial(req)
	if err != nil {
		return nil, err
	}
	setupCtx, err := outgoing(ctx, req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	method, err := resolveMethod(setupCtx, conn, req.Method)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !method.IsStreamingClient() && !method.IsStreamingServer() {
		conn.Close()
		return nil, fmt.Errorf("%s is a unary method; turn Stream off to call it", req.Method)
	}
	streamCtx, cancel := context.WithCancel(context.Background())
	streamCtx, err = outgoing(streamCtx, req)
	if err != nil {
		cancel()
		conn.Close()
		return nil, err
	}
	desc := &grpc.StreamDesc{ServerStreams: method.IsStreamingServer(), ClientStreams: method.IsStreamingClient()}
	stream, err := conn.NewStream(streamCtx, desc, "/"+req.Method)
	if err != nil {
		cancel()
		conn.Close()
		return nil, errors.New(grpcErrText(ctx, err))
	}
	session := &grpcSession{
		events: make(chan Event, eventBuffer), done: make(chan struct{}), cancel: cancel,
		stream: stream, method: method,
	}
	session.emit(Event{Dir: Info, Kind: "open", Text: fmt.Sprintf("Streaming %s on %s", req.Method, req.URL)})
	first := strings.TrimSpace(req.Body)
	if first != "" || !method.IsStreamingClient() {
		if first == "" {
			first = "{}"
		}
		if err := session.sendMessage(first); err != nil {
			cancel()
			conn.Close()
			return nil, err
		}
	}
	if !method.IsStreamingClient() {
		_ = stream.CloseSend() // a failure shows up as the stream's own error
	}
	go session.receive(conn)
	return session, nil
}

// grpcSession is a live gRPC stream shown as a log of events.
type grpcSession struct {
	events chan Event
	done   chan struct{}
	cancel context.CancelFunc
	stream grpc.ClientStream
	method protoreflect.MethodDescriptor

	sendMu sync.Mutex // SendMsg must not run concurrently
	mu     sync.Mutex
	ended  bool
	closed bool // Close was called: the end of the stream is not news
}

func (s *grpcSession) Events() <-chan Event { return s.events }

func (s *grpcSession) emit(event Event) {
	event.Time = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	select {
	case s.events <- event:
	case <-s.done:
	}
}

func (s *grpcSession) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ended {
		s.ended = true
		close(s.events)
	}
}

func (s *grpcSession) isClosed() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.closed }

// receive reads messages until the stream ends, then reports how.
func (s *grpcSession) receive(conn *grpc.ClientConn) {
	defer func() {
		conn.Close()
		s.finish()
	}()
	for {
		message := dynamicpb.NewMessage(s.method.Output())
		err := s.stream.RecvMsg(message)
		if err != nil {
			s.reportEnd(err)
			return
		}
		text, err := protojson.Marshal(message)
		if err != nil {
			s.emit(Event{Dir: Info, Kind: "close", Text: "A message cannot be shown as JSON: " + err.Error()})
			return
		}
		s.emit(Event{Dir: Received, Kind: "message", Text: compactJSON(text), Size: len(text)})
	}
}

// reportEnd logs why the stream ended, unless the user closed it.
func (s *grpcSession) reportEnd(err error) {
	if s.isClosed() {
		return
	}
	if errors.Is(err, io.EOF) {
		s.emit(Event{Dir: Info, Kind: "close", Text: "The server ended the stream"})
		return
	}
	st, _ := status.FromError(err)
	s.emit(Event{Dir: Info, Kind: "close", Code: int(st.Code()), Text: fmt.Sprintf("Stream ended: %s: %s", st.Code(), st.Message())})
}

// Send implements Session: a JSON message of the method's input type, or
// "end" to close this side.
func (s *grpcSession) Send(text string) error {
	if !s.method.IsStreamingClient() {
		return errors.New("this method takes one message, sent when connecting")
	}
	text = strings.TrimSpace(text)
	if text == grpcEndText {
		s.sendMu.Lock()
		defer s.sendMu.Unlock()
		if err := s.stream.CloseSend(); err != nil {
			return err
		}
		s.emit(Event{Dir: Info, Kind: "end", Text: "This side of the stream is closed"})
		return nil
	}
	return s.sendMessage(text)
}

func (s *grpcSession) sendMessage(text string) error {
	message := dynamicpb.NewMessage(s.method.Input())
	if err := protojson.Unmarshal([]byte(text), message); err != nil {
		return fmt.Errorf("the message is not valid %s JSON: %w", s.method.Input().FullName(), err)
	}
	s.emit(Event{Dir: Sent, Kind: "message", Text: compactJSON([]byte(text)), Size: len(text)}) // before the send: an answer may arrive during it
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if err := s.stream.SendMsg(message); err != nil {
		return errors.New(grpcErrText(context.Background(), err))
	}
	return nil
}

func (s *grpcSession) Ping() error { return errors.New("a gRPC stream has no ping") }

func (s *grpcSession) Close() error {
	s.mu.Lock()
	s.closed = true
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.mu.Unlock()
	s.cancel()
	return nil
}
