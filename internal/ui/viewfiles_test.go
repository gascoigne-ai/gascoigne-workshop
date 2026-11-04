package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The status bar template renders exactly what the Go view it replaced did.
func TestStatusBarTemplateMatchesTheGoView(t *testing.T) {
	a, _ := newTestApp(t)
	a.status = "Sent 200 OK in 120ms"
	goView := Row(H(statusH), PadXY(w.Pad, 0), Gap(w.Pad), Bg(w.Frame), Control("statusbar"), BorderTop(w.Border),
		Text(a.status, Small(), Muted(), Ellipsis(), Grow(1)),
		Text(version, Small(), Muted()),
	)
	tplView := a.statusBar()
	Layout(a.ctx, goView, 900, statusH)
	Layout(a.ctx, tplView, 900, statusH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func TestViewsDirReplacesAndReloadsTemplates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "statusbar.vml")
	write := func(src string, mod time.Time) {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(path, mod, mod)
	}
	t0 := time.Now().Add(-time.Hour)
	write(`<row h="28"><text>custom {{status}}</text></row>`, t0)
	a, err := NewApp(Options{ViewsDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a.status = "ok"
	if d := Dump(a.statusBar()); !strings.Contains(d, `"custom ok"`) {
		t.Fatal(d)
	}
	write(`<row h="28"><text>edited</text></row>`, t0.Add(time.Minute))
	if d := Dump(a.statusBar()); !strings.Contains(d, `"edited"`) {
		t.Fatalf("edit not picked up: %s", d)
	}
	write(`<row><blink/></row>`, t0.Add(2*time.Minute))
	if d := Dump(a.statusBar()); !strings.Contains(d, "unknown element <blink>") {
		t.Fatalf("a broken template must show its error: %s", d)
	}
}
