//go:build !release

package e2e

import (
	"cmp"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/collectionstore"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter/historyfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
)

const testRetryDelay = 10 * time.Millisecond

// realAdapters gives opts the real stores, protocols and fonts, as main
// does. It lives in e2e, not in apptest, because apptest is in the app zone
// and the app zone imports no adapter.
func realAdapters(opts app.Options) app.Options {
	opts.HostOS = cmp.Or(opts.HostOS, "linux") // the same shortcut text on every machine
	opts.RetryDelay = cmp.Or(opts.RetryDelay, testRetryDelay)
	opts.History = func(dir string) history.History { return historyfile.Dir(dir) }
	opts.Fonts = fonts.System{}
	opts.Protocols = protocol.Configured
	opts.GRPCMethods = protocol.ListGRPCMethods
	opts.GRPCSchema = protocol.GRPCInputSchema
	opts.Curl = protocol.Curl
	opts.Store = collectionstore.Files{}
	opts.States = statefile.Files{}
	opts.SettingsFiles = settingsfile.Files{}
	return opts
}
