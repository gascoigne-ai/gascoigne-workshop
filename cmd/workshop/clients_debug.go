//go:build !release

package main

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"

// clientConfigs is nil in the default build: it starts from the sample,
// which carries the example server's endpoints, and reads no .workshop
// file, so a stray one cannot hide them.
func clientConfigs() clientconfig.Source { return nil }
