//go:build !release

package main

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"

// startWith is the sample in the default build: it starts with the example
// server's endpoints, and reads no .workshop file, so a stray one cannot hide
// them.
const startWith = collection.StartWithSample
