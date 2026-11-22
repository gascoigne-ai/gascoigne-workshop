//go:build release

package main

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
)

// A release build ships no example endpoints: it starts empty and loads
// the user's workspace or client config.
func TestReleaseBuildLoadsTheUsersDataAndShipsNoSample(t *testing.T) {
	if clientConfigs() == nil {
		t.Error("a release build reads the client config")
	}
	sample := collection.Sample()
	if len(sample.Folders) != 0 || len(sample.Requests) != 0 {
		t.Errorf("release sample has %d folders and %d requests", len(sample.Folders), len(sample.Requests))
	}
}
