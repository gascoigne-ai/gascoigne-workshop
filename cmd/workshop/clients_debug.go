//go:build !release

package main

import cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"

// clientConfigs is nil in the default build: it starts from the sample,
// which carries the example server's endpoints, and reads no .workshop
// file, so a stray one cannot hide them.
func clientConfigs() cport.ClientConfigs { return nil }

// ignoreRememberedWorkspace is true in the default build: an earlier run
// may have saved and remembered another collection, and a run from an IDE
// must still show the example server.
const ignoreRememberedWorkspace = true
