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

type described struct{ view.Base }

func (described) Describe() string { return "glyph=star" }

func TestDumpPrintsWhatADescriberSays(t *testing.T) {
	dump := view.Dump(view.Custom(described{}))
	if first, _, _ := strings.Cut(dump, "\n"); first != "described glyph=star" {
		t.Fatalf("dump %q, want the kind and the description", first)
	}
}

func TestAnElementReportsTheLookItWasGiven(t *testing.T) {
	node := view.Custom(&probe{}, view.W(20), view.H(10), view.Color(core.Accent), view.HoverColor(core.Error))
	if w, h := node.FixedSize(); w != 20 || h != 10 {
		t.Errorf("FixedSize = %d, %d", w, h)
	}
	if got := node.ColorToken(core.Muted); got != core.Accent {
		t.Errorf("ColorToken = %q, want the one given", got)
	}
	if got := node.HoverColorToken(); got != core.Error {
		t.Errorf("HoverColorToken = %q", got)
	}
	bare := view.Custom(&probe{})
	if w, h := bare.FixedSize(); w != 0 || h != 0 {
		t.Errorf("a bare element has no fixed size: %d, %d", w, h)
	}
	if got := bare.ColorToken(core.Muted); got != core.Muted {
		t.Errorf("ColorToken without Color = %q, want the fallback", got)
	}
	if got := bare.HoverColorToken(); got != "" {
		t.Errorf("HoverColorToken without HoverColor = %q", got)
	}
}

// order records what the runtime calls and in which sequence.
type order struct{ calls []string }

func (o *order) note(call string) { o.calls = append(o.calls, call) }

type hooked struct {
	view.Base
	order *order
}

func (h hooked) PreUpdate(*view.El, *core.Ctx)               { h.order.note("pre") }
func (h hooked) Update(*view.El, *core.Ctx)                  { h.order.note("update") }
func (h hooked) Draw(*view.El, *core.Ctx, *ebiten.Image)     { h.order.note("draw") }
func (h hooked) DrawOver(*view.El, *core.Ctx, *ebiten.Image) { h.order.note("over") }

type noter struct {
	view.Base
	order *order
	name  string
}

func (n noter) Update(*view.El, *core.Ctx)              { n.order.note(n.name + " update") }
func (n noter) Draw(*view.El, *core.Ctx, *ebiten.Image) { n.order.note(n.name + " draw") }

func TestPreUpdaterRunsBeforeTheChildrenAndOverDrawerAfterThem(t *testing.T) {
	ctx := newCtx(t)
	rec := &order{}
	root := view.Custom(hooked{order: rec}, view.Custom(noter{order: rec, name: "child"}))
	view.Layout(ctx, root, 100, 100)
	view.Update(ctx, root)
	view.Draw(ctx, root, ebiten.NewImage(100, 100))
	got := strings.Join(rec.calls, ",")
	want := "pre,child update,update,draw,child draw,over"
	if got != want {
		t.Fatalf("calls %s, want %s", got, want)
	}
}

type scrollProbe struct {
	view.Linear
	targets []core.Rect
}

func (p *scrollProbe) ScrollTo(_ *view.El, target core.Rect) { p.targets = append(p.targets, target) }

func TestTabAsksAScrollerToShowTheFocusedElement(t *testing.T) {
	ctx := newCtx(t)
	first, second := &inputState{focused: true}, &inputState{}
	probe := &scrollProbe{}
	target := view.Custom(inputB{state: second}, view.H(20))
	root := view.Custom(probe, view.Custom(inputB{state: first}, view.H(20)), target)
	view.Layout(ctx, root, 200, 100)
	ctx.In = core.Input{X: 5000, Y: 5000, Tab: true} // the second element takes the focus
	view.Update(ctx, root)
	if !second.focused {
		t.Fatal("setup: Tab must move the focus to the second element")
	}
	if len(probe.targets) != 1 || probe.targets[0] != target.Rect() {
		t.Fatalf("ScrollTo got %v, want the second element's rect %+v", probe.targets, target.Rect())
	}
}

func TestAnElementReportsItsPlaceInLayout(t *testing.T) {
	ctx := newCtx(t)
	hidden := view.Custom(&probe{size: 30}, view.Optional()) // dropped: the row is too narrow for it
	floating := view.Custom(&probe{}, view.FloatAt(0, 0))
	plain := view.Custom(&probe{size: 10}, view.KeepInView(), view.PadL(3), view.PadR(4))
	root := view.Custom(view.Linear{Horizontal: true}, hidden, floating, plain)
	view.Layout(ctx, root, 20, 100)
	if hidden.InFlow() || floating.InFlow() || !plain.InFlow() {
		t.Errorf("InFlow: hidden %v floating %v plain %v", hidden.InFlow(), floating.InFlow(), plain.InFlow())
	}
	if hidden.KeepsInView() || !plain.KeepsInView() {
		t.Errorf("KeepsInView: hidden %v plain %v", hidden.KeepsInView(), plain.KeepsInView())
	}
	if inner := plain.Inner(); inner.X != plain.Rect().X+3 || inner.W != plain.Rect().W-7 {
		t.Errorf("Inner %+v inside %+v", inner, plain.Rect())
	}
}

func TestLinearLayoutInLaysChildrenOutInAShiftedRect(t *testing.T) {
	ctx := newCtx(t)
	child := view.Custom(&probe{size: 30})
	shifted := scroller{}
	root := view.Custom(shifted, child)
	view.Layout(ctx, root, 100, 50)
	if got := child.Rect(); got.Y != -40 || got.W != 100 {
		t.Fatalf("child %+v, want it laid out 40 above the box at the box width", got)
	}
	if got := root.Rect(); got != (core.Rect{W: 100, H: 50}) {
		t.Fatalf("the container keeps its own rect, got %+v", got)
	}
}

// scroller lays its children out in a rect 40 pixels higher than its own.
type scroller struct{ view.Linear }

func (s scroller) Layout(e *view.El, c *core.Ctx) {
	r := e.Rect()
	s.LayoutIn(e, c, core.Rect{X: r.X, Y: r.Y - 40, W: r.W, H: r.H + 40})
}

// input is a focusable component written outside view: a state with a
// focus flag, and an element that exposes it.
type inputState struct{ focused bool }

func (s *inputState) HasFocus() bool   { return s.focused }
func (s *inputState) SetFocus(on bool) { s.focused = on }

type inputB struct {
	view.Base
	state   *inputState
	keepTab bool
}

func (b inputB) FocusTarget() view.Focuser { return b.state }
func (b inputB) KeepsTab() bool            { return b.keepTab }

func TestTabMovesFocusBetweenFocusableCustomElements(t *testing.T) {
	ctx := newCtx(t)
	states := []*inputState{{}, {}, {}}
	root := view.Col(
		view.Custom(inputB{state: states[0]}, view.H(10)),
		view.Custom(inputB{state: states[1]}, view.H(10)),
		view.Custom(inputB{state: states[2]}, view.H(10)),
	)
	view.Layout(ctx, root, 100, 100)
	focus := func() (on [3]bool) {
		for i, state := range states {
			on[i] = state.focused
		}
		return on
	}
	tab := func(shift bool) {
		ctx.In = core.Input{X: 5000, Y: 5000, Tab: true, Shift: shift}
		view.Update(ctx, root)
	}

	tab(false)
	if focus() != [3]bool{true, false, false} {
		t.Fatalf("Tab with nothing focused starts at the first: %v", focus())
	}
	tab(false)
	tab(false)
	if focus() != [3]bool{false, false, true} {
		t.Fatalf("two more Tabs reach the last: %v", focus())
	}
	tab(true)
	if focus() != [3]bool{false, true, false} {
		t.Fatalf("Shift+Tab goes back one: %v", focus())
	}
}

func TestOnlyOneCustomElementKeepsTheFocus(t *testing.T) {
	ctx := newCtx(t)
	first, second := &inputState{focused: true}, &inputState{}
	root := view.Col(view.Custom(inputB{state: first}, view.H(10)), view.Custom(inputB{state: second}, view.H(10)))
	view.Layout(ctx, root, 100, 100)
	view.Update(ctx, root) // the first one owns the focus
	second.focused = true  // the app asks for it on the second
	view.Update(ctx, root)
	if first.focused || !second.focused {
		t.Fatalf("the last to ask has the keyboard: first %v second %v", first.focused, second.focused)
	}
}

func TestACustomElementThatKeepsTabKeepsTheFocusOnTab(t *testing.T) {
	ctx := newCtx(t)
	editor, other := &inputState{focused: true}, &inputState{}
	root := view.Col(view.Custom(inputB{state: editor, keepTab: true}, view.H(10)), view.Custom(inputB{state: other}, view.H(10)))
	view.Layout(ctx, root, 100, 100)
	ctx.In = core.Input{X: 5000, Y: 5000, Tab: true}
	view.Update(ctx, root)
	if !editor.focused || other.focused {
		t.Fatalf("an editor that uses Tab keeps the focus: editor %v other %v", editor.focused, other.focused)
	}
}

func TestControlNameIsTheStyleGivenWithControl(t *testing.T) {
	if got := view.Custom(&probe{}, view.Control("field")).ControlName(); got != "field" {
		t.Fatalf("ControlName = %q, want the control given", got)
	}
	if got := view.Custom(&probe{}).ControlName(); got != "" {
		t.Fatalf("ControlName without Control = %q", got)
	}
}
