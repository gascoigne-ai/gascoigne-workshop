package window

import (
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// withoutPorts is a window started with the three shared stores and nothing
// else: every adapter is left to the null ones.
func withoutPorts(t *testing.T, ports Ports) *Component {
	t.Helper()
	window, err := New(Stores{Tabs: tabs.NewSet(), Sends: sends.New(), Bus: bus.New(), Ports: ports})
	if err != nil {
		t.Fatal(err)
	}
	return window
}

func TestNewFillsMissingPorts(t *testing.T) {
	window := withoutPorts(t, Ports{})
	for name, nullOf := range map[string]any{
		"collection store": window.store, "file dialog": window.files, "folders": window.folders,
		"fonts": window.fonts, "previews": window.previews, "settings files": window.settingFiles, "states": window.states,
	} {
		if nullOf == nil {
			t.Errorf("the %s port stayed nil, every call to it would panic", name)
		}
	}
	if window.now == nil {
		t.Error("no clock: the wall clock is the default")
	}
	if window.retryDelay != defaultRetryDelay {
		t.Errorf("retry delay %v, want the default %v", window.retryDelay, defaultRetryDelay)
	}
}

func TestNewKeepsThePortsItIsGiven(t *testing.T) {
	want := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	store := noStore{}
	window := withoutPorts(t, Ports{Now: func() time.Time { return want }, Store: store})
	if got := window.Now(); !got.Equal(want) {
		t.Fatalf("now = %v, want the given clock %v", got, want)
	}
}
