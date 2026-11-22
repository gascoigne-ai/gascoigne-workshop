package settingsui

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// failingStore is the real files with a write that always fails.
type failingStore struct{ settingsfile.Files }

func (failingStore) WriteText(string, []byte) error { return errors.New("disk full") }

func rawHostWithFailingWrites(t *testing.T) (*fakeHost, *SettingsDialog) {
	t.Helper()
	host := newHost(t)
	if err := os.WriteFile(host.path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	host.store = failingStore{}
	var d SettingsDialog
	d.OpenRaw(host)
	return host, &d
}

func TestAFailingRawWriteToastsAndDoesNotStamp(t *testing.T) {
	host, d := rawHostWithFailingWrites(t)
	host.current.AutoSave = true
	d.RawText.Text = "{\n  \"layout\": \"side\"\n}"
	d.RawEdited(host)
	if len(host.toasts) != 1 || !strings.Contains(host.toasts[0], "disk full") {
		t.Fatalf("toasts %q", host.toasts)
	}
	if d.stamp.Exists {
		t.Fatalf("the failed write was stamped: %+v", d.stamp)
	}
}

func TestAFailingSaveKeepsTheRawTextPending(t *testing.T) {
	host, d := rawHostWithFailingWrites(t)
	d.rawPending = true
	d.Save(host)
	if !d.rawPending || d.stamp.Exists || len(host.toasts) != 1 {
		t.Fatalf("pending %v stamp %+v toasts %q", d.rawPending, d.stamp, host.toasts)
	}
	if host.unsaved {
		t.Fatal("a failed save marked the file as saved")
	}
}

func TestASuccessfulSaveOfRawTextStampsAndClearsPending(t *testing.T) {
	host := newHost(t)
	var d SettingsDialog
	d.OpenRaw(host)
	d.RawText.Text = "{\n  \"layout\": \"side\"\n}"
	d.rawPending = true
	host.unsaved = true
	d.Save(host)
	if d.rawPending || host.unsaved || !d.stamp.Exists || len(host.toasts) != 0 {
		t.Fatalf("pending %v unsaved %v stamp %+v toasts %q", d.rawPending, host.unsaved, d.stamp, host.toasts)
	}
}

func TestANumberOutsideItsRangeSendsNoChange(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	*d.Input(host, "timeoutSec") = NewTextField("0")
	d.Edited(host, "timeoutSec")
	if host.changes != 0 || d.Note("timeoutSec") == "" {
		t.Fatalf("changes %d note %q", host.changes, d.Note("timeoutSec"))
	}
	*d.Input(host, "timeoutSec") = NewTextField("20")
	d.Edited(host, "timeoutSec")
	if host.changes != 1 || host.current.TimeoutSec != 20 {
		t.Fatalf("changes %d timeout %d", host.changes, host.current.TimeoutSec)
	}
}
