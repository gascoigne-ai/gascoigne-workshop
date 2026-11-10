package proto

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// The protocol contracts live in request/port; the adapters keep using
// their names here.
type (
	Protocol        = port.Protocol
	Registry        = port.Registry
	Direction       = port.Direction
	Event           = port.Event
	Session         = port.Session
	BinarySender    = port.BinarySender
	Requester       = port.Requester
	Streamer        = port.Streamer
	Sampler         = port.Sampler
	SamplingRequest = port.SamplingRequest
	SamplingMessage = port.SamplingMessage
	SamplingResult  = port.SamplingResult
)

// Event directions.
const (
	Info     = port.Info
	Sent     = port.Sent
	Received = port.Received
)

// Messages in Response.Err for the two ways a request can be stopped.
const (
	ErrCancelled = model.ErrCancelled
	ErrTimedOut  = model.ErrTimedOut
)
