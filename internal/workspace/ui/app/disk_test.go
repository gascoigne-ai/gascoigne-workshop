package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
)

// The file stores the tests read and write real files through.
var (
	storeDisk    store.Files
	settingsDisk settingsfile.Files
	stateDisk    statefile.Files
)
