//go:build release

package main

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/clientfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"
)

// clientConfigs is the release build's source of collections and servers:
// the .workshop file in the user config directory, overridden entry by
// entry by the one in the working directory.
func clientConfigs() clientconfig.Source {
	return clientfile.Files{GlobalDir: clientfile.DefaultGlobalDir(), LocalDir: "."}
}
