//go:build release

package main

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"

// startWith is nothing in the release build: it opens what the user opened
// before, else no collection. It reads no .workshop file and ships no example
// endpoints, so a first start is the empty "No collection is open" screen.
const startWith = collection.StartWithNothing
