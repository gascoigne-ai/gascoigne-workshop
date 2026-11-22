// Package history is the history feature's model: the responses remembered
// per request, the log of sent requests, and the ports that keep them.
// The disk adapter is in `adapter`, the Sent list's view in `ui/sentlog`.
package history

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// SentLimit is how many sent requests the workspace log keeps.
const SentLimit = 500

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

// Sent is one request that was sent: enough to list it and to open the
// request again.
type Sent struct {
	At time.Time
	// Collection is the folder of the request's collection; "" for one not
	// saved and in rows written when one collection was open at a time,
	// which then name the first.
	Collection string
	Path       string // the request's path in the collection tree
	Protocol   request.Protocol
	Method     string
	URL        string
	Status     int // 0 when no response came
}

// SentStore keeps the log of sent requests between runs.
type SentStore interface {
	// Append adds one sent request to the stored log.
	Append(sent Sent) error
	// Load returns the stored log, newest first.
	Load() ([]Sent, error)
}
