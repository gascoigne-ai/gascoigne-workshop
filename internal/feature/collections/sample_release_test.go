//go:build release

package collection

import "testing"

func TestReleaseSampleIsEmpty(t *testing.T) {
	sample := Sample()
	if len(sample.Folders) != 0 || len(sample.Requests) != 0 {
		t.Fatalf("release sample has %d folders and %d requests, want none", len(sample.Folders), len(sample.Requests))
	}
}
