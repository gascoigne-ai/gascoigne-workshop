package proto

import (
	"crypto/tls"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// tracer records when each phase of a request started and ended.
type tracer struct {
	mu                                     sync.Mutex
	dnsStart, dnsDone, connStart, connDone time.Time
	tlsStart, tlsDone, gotFirstByte        time.Time
}

func (t *tracer) trace() *httptrace.ClientTrace {
	at := func(dst *time.Time) { t.mu.Lock(); *dst = time.Now(); t.mu.Unlock() }
	return &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { at(&t.dnsStart) },
		DNSDone:              func(httptrace.DNSDoneInfo) { at(&t.dnsDone) },
		ConnectStart:         func(_, _ string) { at(&t.connStart) },
		ConnectDone:          func(_, _ string, _ error) { at(&t.connDone) },
		TLSHandshakeStart:    func() { at(&t.tlsStart) },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { at(&t.tlsDone) },
		GotFirstResponseByte: func() { at(&t.gotFirstByte) },
	}
}

func span(from, to time.Time) time.Duration {
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return 0
	}
	return to.Sub(from)
}

// timing turns the recorded moments into phases: sent at start, headers
// read at headers, body read at end. Connect excludes the TLS handshake,
// which is reported on its own.
func (t *tracer) timing(start, headers, end time.Time) model.Timing {
	t.mu.Lock()
	defer t.mu.Unlock()
	connect := span(t.connStart, t.connDone)
	return model.Timing{
		DNS:       span(t.dnsStart, t.dnsDone),
		Connect:   connect,
		TLS:       span(t.tlsStart, t.tlsDone),
		FirstByte: span(start, t.gotFirstByte),
		Transfer:  span(headers, end),
	}
}
