package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartProfileWritesTheCPUProfileOnStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.prof")
	stop, err := startProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("the profile file must exist and hold data, got %v %v", info, err)
	}
}

func TestStartProfileWithNoPathDoesNothing(t *testing.T) {
	stop, err := startProfile("")
	if err != nil {
		t.Fatal(err)
	}
	stop()
}

func TestStartProfileReportsAnUnwritablePath(t *testing.T) {
	if _, err := startProfile(filepath.Join(t.TempDir(), "missing", "cpu.prof")); err == nil {
		t.Fatal("an unwritable path must be an error")
	}
}
