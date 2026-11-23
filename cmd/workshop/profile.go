package main

import (
	"fmt"
	"os"
	"runtime/pprof"
)

// startProfile writes a CPU profile to path until the returned stop is
// called; an empty path profiles nothing. It is a flag, not an HTTP
// endpoint, so a run opens no port: the profile is read with
// `go tool pprof`.
func startProfile(path string) (stop func(), err error) {
	if path == "" {
		return func() {}, nil
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create cpu profile %s: %w", path, err)
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("start cpu profile: %w", err)
	}
	return func() {
		pprof.StopCPUProfile()
		file.Close()
	}, nil
}
