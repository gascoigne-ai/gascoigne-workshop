package app

import (
	"os"
	"testing"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// TestMain pins the platform so shortcut text and pinned layouts read the
// same on every machine.
func TestMain(runner *testing.M) {
	w.HostOS = "linux"
	os.Exit(runner.Run())
}
