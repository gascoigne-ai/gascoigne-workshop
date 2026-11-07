package proto

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// URL schemes of gRPC requests: the first talks without TLS, the second
// with it (and honors Request.InsecureTLS). A bare host:port is the first.
const (
	grpcScheme  = "grpc://"
	grpcsScheme = "grpcs://"
)

// GRPC makes unary gRPC calls with JSON messages. Request.URL is the
// server, Request.Method is "package.Service/Method" and Request.Body is
// the message as JSON. The message types come from the server's
// reflection, so no .proto files are needed.
//
// The answer maps to a response like an HTTP one: a call that reached the
// server gives a Status (200 for OK, else the usual HTTP equivalent of
// the gRPC code, see grpcHTTPStatus) and the gRPC code name as
// StatusText; one that could not be made gives Response.Err.
type GRPC struct{}

// grpcHTTPStatus is the HTTP status each gRPC code is shown as, the
// mapping gRPC gateways use. Codes not listed show as 500.
var grpcHTTPStatus = map[codes.Code]int{
	codes.OK:                 http.StatusOK,
	codes.Canceled:           499,
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.FailedPrecondition: http.StatusBadRequest,
	codes.Aborted:            http.StatusConflict,
	codes.OutOfRange:         http.StatusBadRequest,
	codes.Unimplemented:      http.StatusNotImplemented,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.Unauthenticated:    http.StatusUnauthorized,
}

// dial opens a client connection to the server of req. It connects
// lazily: an unreachable server shows on the first call.
func dial(req model.Request) (*grpc.ClientConn, error) {
	target, creds := req.URL, credentials.TransportCredentials(insecure.NewCredentials())
	switch {
	case strings.HasPrefix(target, grpcsScheme):
		target = strings.TrimPrefix(target, grpcsScheme)
		creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: req.InsecureTLS, MinVersion: tls.VersionTLS12}) //nolint:gosec // the user's explicit "insecure TLS" option
	case strings.HasPrefix(target, grpcScheme):
		target = strings.TrimPrefix(target, grpcScheme)
	}
	if target == "" {
		return nil, errors.New("the URL names no gRPC server, for example grpc://localhost:50051")
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", target, err)
	}
	return conn, nil
}

// outgoing is the metadata of a call: the enabled headers and the
// Authorization the Auth pane makes, lowercased as gRPC requires.
func outgoing(ctx context.Context, req model.Request) (context.Context, error) {
	pairs := map[string]string{}
	for _, header := range req.Headers {
		if header.Enabled && header.Name != "" {
			pairs[strings.ToLower(header.Name)] = header.Value
		}
	}
	apply, ok := authAppliers[req.Auth.Type]
	if !ok {
		return nil, fmt.Errorf("unknown auth type %q", req.Auth.Type)
	}
	carrier := &http.Request{Header: http.Header{}}
	apply(req.Auth, carrier)
	for name, values := range carrier.Header {
		pairs[strings.ToLower(name)] = values[0]
	}
	md := metadata.New(pairs)
	return metadata.NewOutgoingContext(ctx, md), nil
}

// ListGRPCMethods lists the methods of the server as
// "package.Service/Method", asking its reflection service. The
// reflection service itself is left out.
func ListGRPCMethods(ctx context.Context, req model.Request) ([]string, error) {
	conn, err := dial(req)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	ctx, err = outgoing(ctx, req)
	if err != nil {
		return nil, err
	}
	services, err := listServices(ctx, conn)
	if err != nil {
		return nil, err
	}
	var methods []string
	for _, name := range services {
		if strings.HasPrefix(name, "grpc.reflection.") {
			continue
		}
		files, err := loadSymbol(ctx, conn, name)
		if err != nil {
			return nil, err
		}
		service, err := findService(files, name)
		if err != nil {
			return nil, err
		}
		for i := 0; i < service.Methods().Len(); i++ {
			methods = append(methods, name+"/"+string(service.Methods().Get(i).Name()))
		}
	}
	sort.Strings(methods)
	return methods, nil
}

// Send makes the unary call; see GRPC.
func (GRPC) Send(ctx context.Context, req model.Request) (model.Response, error) {
	start := time.Now()
	fail := func(err error) (model.Response, error) {
		return model.Response{Err: grpcErrText(ctx, err), Duration: time.Since(start)}, nil
	}
	conn, err := dial(req)
	if err != nil {
		return fail(err)
	}
	defer conn.Close()
	ctx, err = outgoing(ctx, req)
	if err != nil {
		return fail(err)
	}
	method, err := resolveMethod(ctx, conn, req.Method)
	if err != nil {
		return fail(err)
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return fail(fmt.Errorf("%s is a streaming method; only unary calls are supported", req.Method))
	}
	input := dynamicpb.NewMessage(method.Input())
	body := strings.TrimSpace(req.Body)
	if body == "" {
		body = "{}"
	}
	if err := protojson.Unmarshal([]byte(body), input); err != nil {
		return fail(fmt.Errorf("the message is not valid %s JSON: %w", method.Input().FullName(), err))
	}
	output := dynamicpb.NewMessage(method.Output())
	var header, trailer metadata.MD
	err = conn.Invoke(ctx, "/"+req.Method, input, output, grpc.Header(&header), grpc.Trailer(&trailer))
	return grpcResponse(output, err, header, trailer, time.Since(start))
}

// grpcResponse turns the result of a call into a response.
func grpcResponse(output *dynamicpb.Message, callErr error, header, trailer metadata.MD, took time.Duration) (model.Response, error) {
	response := model.Response{Duration: took, Headers: metadataRows(header, trailer)}
	code, text := codes.OK, []byte(nil)
	if callErr != nil {
		st, _ := status.FromError(callErr)
		code, text = st.Code(), []byte(st.Message())
	} else {
		// protojson randomizes its spacing on purpose; indenting its compact
		// output ourselves gives the same text every time.
		compact, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(output)
		if err != nil {
			return model.Response{Err: "the answer cannot be shown as JSON: " + err.Error(), Duration: took}, nil
		}
		var indented bytes.Buffer
		if err := json.Indent(&indented, compact, "", "  "); err != nil {
			return model.Response{Err: "the answer cannot be shown as JSON: " + err.Error(), Duration: took}, nil
		}
		text = indented.Bytes()
	}
	httpStatus, known := grpcHTTPStatus[code]
	if !known {
		httpStatus = http.StatusInternalServerError
	}
	response.Status, response.StatusText = httpStatus, code.String()
	response.Body, response.Size = text, len(text)
	return response, nil
}

// metadataRows lists header and trailer metadata as response headers.
func metadataRows(header, trailer metadata.MD) []model.KV {
	var rows []model.KV
	for _, md := range []metadata.MD{header, trailer} {
		for name, values := range md {
			for _, value := range values {
				rows = append(rows, model.KV{Name: name, Value: value, Enabled: true})
			}
		}
	}
	sortKV(rows)
	return rows
}

// grpcErrText is the message of a call that could not be made, with the
// two ways a request is stopped named as REST names them.
func grpcErrText(ctx context.Context, err error) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
		return ErrTimedOut
	case errors.Is(ctx.Err(), context.Canceled) || status.Code(err) == codes.Canceled:
		return ErrCancelled
	}
	return err.Error()
}

// resolveMethod finds the descriptor of "package.Service/Method" through
// the server's reflection.
func resolveMethod(ctx context.Context, conn *grpc.ClientConn, name string) (protoreflect.MethodDescriptor, error) {
	serviceName, methodName, ok := strings.Cut(name, "/")
	if !ok || serviceName == "" || methodName == "" {
		return nil, fmt.Errorf("the method %q is not package.Service/Method", name)
	}
	files, err := loadSymbol(ctx, conn, serviceName)
	if err != nil {
		return nil, err
	}
	service, err := findService(files, serviceName)
	if err != nil {
		return nil, err
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, fmt.Errorf("%s has no method %s", serviceName, methodName)
	}
	return method, nil
}
