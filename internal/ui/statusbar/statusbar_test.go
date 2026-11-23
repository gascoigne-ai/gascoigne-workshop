package statusbar

import (
	"image/color"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

type fakeDeps struct {
	status   string
	logOpens int
	welcomes int
}

func (fake *fakeDeps) Status() string { return fake.status }
func (fake *fakeDeps) Open()          { fake.logOpens++ }
func (fake *fakeDeps) OpenWelcome()   { fake.welcomes++ }

func newBar(t *testing.T, fake *fakeDeps) (*core.Ctx, *Component) {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, New(Deps{Status: fake, Log: fake, Welcome: fake, Version: "v9.9"})
}

// laidOut is the bar in a window 900 wide.
func laidOut(ctx *core.Ctx, bar *Component) Node {
	root := bar.View(ctx)
	Layout(ctx, root, 900, sizes.StatusH)
	return root
}

// clickOn clicks the middle of the first element whose own Dump line has part.
func clickOn(t *testing.T, ctx *core.Ctx, root Node, part string) {
	t.Helper()
	var find func(node Node) Node
	find = func(node Node) Node {
		line, _, _ := strings.Cut(Dump(node), "\n")
		if strings.Contains(line, part) {
			return node
		}
		for _, child := range node.Children() {
			if found := find(child); found != nil {
				return found
			}
		}
		return nil
	}
	node := find(root)
	if node == nil {
		t.Fatalf("no element with %s in:\n%s", part, Dump(root))
	}
	rect := node.Rect()
	ctx.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	Update(ctx, root)
}

func TestTheBarShowsTheStatusAndTheVersion(t *testing.T) {
	ctx, bar := newBar(t, &fakeDeps{status: "Sent 200 OK"})
	dump := Dump(laidOut(ctx, bar))
	for _, want := range []string{"Sent 200 OK", "v9.9"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}

func TestTheLogButtonOpensTheLog(t *testing.T) {
	fake := &fakeDeps{}
	ctx, bar := newBar(t, fake)
	clickOn(t, ctx, laidOut(ctx, bar), `text="Log"`)
	if fake.logOpens != 1 {
		t.Fatalf("log opened %d times, want 1", fake.logOpens)
	}
}

func TestTheVersionOpensTheWelcome(t *testing.T) {
	fake := &fakeDeps{}
	ctx, bar := newBar(t, fake)
	clickOn(t, ctx, laidOut(ctx, bar), `text="v9.9"`)
	if fake.welcomes != 1 {
		t.Fatalf("welcome opened %d times, want 1", fake.welcomes)
	}
}
