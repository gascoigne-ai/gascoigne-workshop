package overlays

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// statCounter is a settings store that counts how often the file is looked at.
type statCounter struct {
	settings.SettingsStore
	stats int
}

func (counter *statCounter) Stat(string) settings.FileStamp {
	counter.stats++
	return settings.FileStamp{}
}

func TestTickPollsTheSettingsEveryPollTicks(t *testing.T) {
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	window := &fakeWindow{}
	counter := &statCounter{}
	over := New(Deps{
		Ctx: ctx, Window: window, Clock: fakeClock{}, Desktop: noDesktop{}, Folders: homeFolders{}, Bus: fakeBus{},
		Settings: settingsdialog.Deps{
			Window: settingsWindow{window}, Settings: &fakeSettings{current: settings.Default()},
			Path: "settings.json", Store: counter,
		},
	})
	polled := 0
	for tick := 1; tick <= 3*settingsdialog.PollTicks; tick++ {
		ctx.Tick = tick
		before := counter.stats
		changed := over.Tick()
		onPoll := tick%settingsdialog.PollTicks == 0
		if changed != onPoll {
			t.Errorf("tick %d: Tick() = %v, want %v (the view is stale on a poll)", tick, changed, onPoll)
		}
		if (counter.stats > before) != onPoll {
			t.Errorf("tick %d: looked at the file = %v, want %v", tick, counter.stats > before, onPoll)
		}
		if onPoll {
			polled++
		}
	}
	if polled != 3 {
		t.Fatalf("polled %d times, want 3", polled)
	}
}
