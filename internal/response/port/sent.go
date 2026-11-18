// Package port holds what the response domain needs from the outside: the
// store of sent requests and the types that cross it.
package port

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// SentLimit is how many sent requests the workspace log keeps.
const SentLimit = 500

// Sent is one request that was sent: enough to list it and to open the
// request again.
type Sent struct {
	At time.Time
	// Collection is the folder of the request's collection; "" for one not
	// saved and in rows written when one collection was open at a time,
	// which then name the first.
	Collection string
	Path       string // the request's path in the collection tree
	Protocol   model.Protocol
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
