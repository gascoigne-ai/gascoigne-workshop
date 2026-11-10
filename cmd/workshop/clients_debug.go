//go:build !release

package main

import cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"

// clientConfigs is nil in the default build: it starts from the sample,
// which carries the example server's endpoints, and reads no .workshop
// file, so a stray one cannot hide them.
func clientConfigs() cport.ClientConfigs { return nil }
