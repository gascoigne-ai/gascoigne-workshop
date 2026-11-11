package view_test

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// probe is a component written outside the view package: it wants a size,
// and counts the frames the runtime gives it.
type probe struct {
	view.Base
	size                 int
	updates, draws, laid int
}

func (p *probe) Measure(*view.El, *core.Ctx, int) (int, int) { return p.size, 20 }
func (p *probe) Layout(*view.El, *core.Ctx)                  { p.laid++ }
func (p *probe) Update(*view.El, *core.Ctx)                  { p.updates++ }
func (p *probe) Draw(*view.El, *core.Ctx, *ebiten.Image)     { p.draws++ }

// probeSize is an option of the component: it reaches the behavior through
// the element, as a widget's own options do.
func probeSize(size int) view.Option {
	return func(e *view.El) { e.Behavior().(*probe).size = size }
}

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestCustomBehaviorIsMeasuredLaidOutUpdatedAndDrawn(t *testing.T) {
	ctx := newCtx(t)
	behavior := &probe{size: 30}
	node := view.Custom(behavior)
	root := view.Row(view.AlignItems(view.AlignStart), node)

	view.Layout(ctx, root, 200, 100)
	if got := node.Rect(); got.W != 30 || got.H != 20 {
		t.Fatalf("size from Measure: %+v, want 30x20", got)
	}
	if behavior.laid == 0 {
		t.Fatal("Layout was not called")
	}
	view.Update(ctx, root)
	view.Draw(ctx, root, ebiten.NewImage(200, 100))
	if behavior.updates != 1 || behavior.draws != 1 {
		t.Fatalf("updates %d draws %d, want 1 and 1", behavior.updates, behavior.draws)
	}
}

func TestOptionsOfACustomElementConfigureItsBehavior(t *testing.T) {
	ctx := newCtx(t)
	node := view.Custom(&probe{size: 30}, probeSize(50))
	view.Layout(ctx, view.Row(view.AlignItems(view.AlignStart), node), 200, 100)
	if got := node.Rect().W; got != 50 {
		t.Fatalf("width %d, want 50 from the option", got)
	}
}

func TestBehaviorIsNilForBuiltInElements(t *testing.T) {
	if view.Row().Behavior() != nil {
		t.Fatal("a Row has no custom behavior")
	}
	behavior := &probe{}
	if view.Custom(behavior).Behavior() != view.Behavior(behavior) {
		t.Fatal("Behavior returns what Custom was given")
	}
}

func TestDumpNamesACustomElementByItsKindWithoutPackage(t *testing.T) {
	if dump := view.Dump(view.Custom(&probe{})); !strings.HasPrefix(dump, "probe") {
		t.Fatalf("dump %q, want it to start with the bare kind", dump)
	}
}

func TestLinearLaysCustomChildrenOutLikeColAndRow(t *testing.T) {
	tests := []struct {
		name       string
		linear     view.Linear
		reference  func(...view.Item) *view.El
		wantSecond core.Rect
	}{
		{"vertical is a Col", view.Linear{}, view.Col, core.Rect{X: 0, Y: 20, W: 200, H: 20}},
		{"horizontal is a Row", view.Linear{Horizontal: true}, view.Row, core.Rect{X: 30, Y: 0, W: 30, H: 100}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newCtx(t)
			second := view.Custom(&probe{size: 30})
			custom := view.Custom(test.linear, view.Custom(&probe{size: 30}), second)
			view.Layout(ctx, custom, 200, 100)
			if got := second.Rect(); got != test.wantSecond {
				t.Fatalf("second child at %+v, want %+v", got, test.wantSecond)
			}

			reference := test.reference(view.Custom(&probe{size: 30}), view.Custom(&probe{size: 30}))
			view.Layout(ctx, reference, 200, 100)
			if got := reference.Children()[1].Rect(); got != test.wantSecond {
				t.Fatalf("the built-in container puts it at %+v, the test expects %+v", got, test.wantSecond)
			}
		})
	}
}

// halves is a container written outside view: it puts its two children
// side by side, the first on the left half.
type halves struct{ view.Base }

func (halves) Layout(e *view.El, c *core.Ctx) {
	r := e.Rect()
	kids := e.Children()
	kids[0].Place(c, core.Rect{X: r.X, Y: r.Y, W: r.W / 2, H: r.H})
	kids[1].Place(c, core.Rect{X: r.X + r.W/2, Y: r.Y, W: r.W - r.W/2, H: r.H})
}

func TestPlaceSetsAChildsRectAndLaysItOut(t *testing.T) {
	ctx := newCtx(t)
	inner := &probe{size: 10}
	nested := view.Custom(inner)
	left := view.Custom(halves{}, nested, view.Custom(&probe{}))
	right := view.Custom(&probe{})
	root := view.Custom(halves{}, left, right)

	view.Layout(ctx, root, 400, 100)
	if got := left.Rect(); got != (core.Rect{W: 200, H: 100}) {
		t.Fatalf("left half %+v", got)
	}
	if got := right.Rect(); got != (core.Rect{X: 200, W: 200, H: 100}) {
		t.Fatalf("right half %+v", got)
	}
	if got := nested.Rect(); got != (core.Rect{W: 100, H: 100}) {
		t.Fatalf("a placed child lays its own children out too: %+v", got)
	}
}
