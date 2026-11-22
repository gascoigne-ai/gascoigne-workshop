package settings

// SystemInfo reports what the program runs on: the system, the display and
// the GPU rendering.
type SystemInfo interface {
	// Read is the report as it is now; the numbers that change (frame rate,
	// memory) are read at each call.
	Read() Report
}
