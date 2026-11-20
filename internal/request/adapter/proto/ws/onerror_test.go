package ws

import (
	"bufio"
	"net"
	"testing"
	"time"
)

// brokenPeer is a client Conn whose peer is gone, so every write fails;
// the errors it reports land on the returned channel.
func brokenPeer(t *testing.T) (*Conn, net.Conn, <-chan error) {
	t.Helper()
	local, remote := net.Pipe()
	t.Cleanup(func() { local.Close(); remote.Close() })
	reported := make(chan error, 4)
	conn := &Conn{nc: local, br: bufio.NewReader(local), client: true, max: DefaultMaxMessage,
		onError: func(err error) { reported <- err }}
	return conn, remote, reported
}

func expectReported(t *testing.T, reported <-chan error) {
	t.Helper()
	select {
	case err := <-reported:
		if err == nil {
			t.Fatal("reported a nil error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the failed write was not reported")
	}
}

func TestAFailedCloseFrameIsReported(t *testing.T) {
	conn, remote, reported := brokenPeer(t)
	remote.Close()
	conn.Close(CloseNormal, "")
	expectReported(t, reported)
}

func TestAFailedPongIsReported(t *testing.T) {
	conn, remote, reported := brokenPeer(t)
	go func() {
		remote.Write([]byte{0x80 | byte(Ping), 0}) // an unmasked ping with no payload, as a server sends it
		remote.Close()
	}()
	if m, err := conn.Read(); err != nil || m.Type != Ping {
		t.Fatalf("got %+v, %v", m, err)
	}
	expectReported(t, reported)
}
