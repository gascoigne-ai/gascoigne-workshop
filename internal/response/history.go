package response

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Entry is one remembered response.
type Entry struct {
	At         time.Time
	Status     int
	StatusText string
	Duration   time.Duration
	Headers    []request.KV
	Body       []byte
}

// History keeps the responses of each request and the log of sent requests
// in one folder.
type History interface {
	// Load reads the entries saved for key (a request's path), newest first.
	Load(key string) ([]Entry, error)
	// Save replaces the entries saved for key.
	Save(key string, entries []Entry) error
	// Sent is the sent-request log kept beside the responses.
	Sent() SentStore
}
