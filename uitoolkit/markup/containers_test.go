package markup

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// sameLayout fails unless the template and the Go view dump the same once
// both are laid out in a window of width by height.
func sameLayout(t *testing.T, ctx *core.Ctx, template, goView view.Node, width, height int) {
	t.Helper()
	view.Layout(ctx, template, width, height)
	view.Layout(ctx, goView, width, height)
	if got, want := view.Dump(template), view.Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

// buildError is the error Build gives for src and scope; src must parse.
func buildError(t *testing.T, src string, scope Scope) error {
	t.Helper()
	template, err := Parse("t.vml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = template.Build(scope)
	return err
}

func TestScrollElementsMatchTheGoScrollers(t *testing.T) {
	ctx := testCtx(t)
	column, strip := &widget.ScrollState{Y: 30}, &widget.ScrollState{}
	template := build(t, `
		<col>
		  <scroll state="column" h="100" gap="4"><row h="60"/><row h="60"/></scroll>
		  <hscroll state="strip" w="120" gap="2"><row w="80"/><row w="80" keep-in-view/></hscroll>
		</col>`, Scope{"column": column, "strip": strip})
	goView := view.Col(
		widget.Scroll(column, view.H(100), view.Gap(4), view.Row(view.H(60)), view.Row(view.H(60))),
		widget.HScroll(strip, view.W(120), view.Gap(2), view.Row(view.W(80)), view.Row(view.W(80), view.KeepInView())),
	)
	sameLayout(t, ctx, template, goView, 300, 200)
}

func TestScrollElementScrollsTheStateOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	column := &widget.ScrollState{}
	template := build(t, `<col><scroll state="column" h="100"><row h="300"/></scroll></col>`, Scope{"column": column})
	view.Layout(ctx, template, 200, 100)
	ctx.In = core.Input{X: 50, Y: 50, Wheel: -1}
	view.Update(ctx, template)
	if column.Y <= 0 {
		t.Fatalf("a wheel turn down left the scope's state at Y = %d", column.Y)
	}
}

func TestKeepInViewScrollsTheStripToTheFlaggedChild(t *testing.T) {
	ctx := testCtx(t)
	strip := &widget.ScrollState{}
	template := build(t, `<row><hscroll state="strip" w="100"><row w="90"/><row w="90"/><row w="90" keep-in-view/></hscroll></row>`, Scope{"strip": strip})
	view.Layout(ctx, template, 300, 40)
	view.Update(ctx, template)
	if strip.X <= 0 {
		t.Fatalf("the strip stayed at X = %d, want it scrolled to the third child", strip.X)
	}
}

func TestScrollStateOfTheWrongTypeNamesTheTypeItWants(t *testing.T) {
	err := buildError(t, `<scroll state="column"/>`, Scope{"column": widget.ScrollState{}})
	if err == nil || !strings.Contains(err.Error(), "*widget.ScrollState") {
		t.Errorf("error = %v, want it to say column is not a *widget.ScrollState", err)
	}
}

// A nil *widget.ScrollState in the scope passed the type check, and the
// scroll panicked at layout, far from the template that bound it.
func TestNilStateInTheScopeIsABuildErrorNamingTheEntry(t *testing.T) {
	tests := []struct {
		name, src string
		scope     Scope
		want      string
	}{
		{"scroll", `<scroll state="list"/>`, Scope{"list": (*widget.ScrollState)(nil)}, "*widget.ScrollState"},
		{"hscroll", `<hscroll state="list"/>`, Scope{"list": (*widget.ScrollState)(nil)}, "*widget.ScrollState"},
		{"split", `<split state="list" default="10"/>`, Scope{"list": (*widget.SplitState)(nil)}, "*widget.SplitState"},
		{"scroll-tabs", `<scroll-tabs state="list" items="tabs" active="0"/>`,
			Scope{"list": (*widget.ScrollState)(nil), "tabs": []widget.TabItem{}}, "*widget.ScrollState"},
		{"message-dialog", `<message-dialog state="list"/>`, Scope{"list": (*widget.DialogState)(nil)}, "*widget.DialogState"},
		{"menu state", `<menu ctx="ctx" state="list"/>`, Scope{"ctx": testCtx(t), "list": (*widget.MenuState)(nil)}, "*widget.MenuState"},
		{"menu ctx", `<menu ctx="list" state="menu"/>`, Scope{"list": (*core.Ctx)(nil), "menu": menuState()}, "*core.Ctx"},
		{"toasts", `<toasts stack="list" width="8" height="6" bottom-inset="2"/>`, Scope{"list": (*widget.ToastStack)(nil)}, "*widget.ToastStack"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := buildErrorOrPanic(t, test.src, test.scope)
			if err == nil || !strings.Contains(err.Error(), `"list"`) || !strings.Contains(err.Error(), "nil "+test.want) {
				t.Errorf("error = %v, want it to say list is a nil %s", err, test.want)
			}
		})
	}
}

// buildErrorOrPanic is buildError that also reports a panic of the build
// as an error, so one bad case does not stop the others.
func buildErrorOrPanic(t *testing.T, src string, scope Scope) (err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("build panicked: %v", recovered)
		}
	}()
	return buildError(t, src, scope)
}

func TestScrollWithoutStateIsAParseError(t *testing.T) {
	_, err := Parse("t.vml", []byte(`<scroll h="10"/>`))
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Errorf("error = %v, want it to say <scroll> needs state", err)
	}
}

func TestSplitElementMatchesTheGoSplit(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		goView     func(state *widget.SplitState, first, second view.Node) view.Node
	}{
		{"pixels", `default="200"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 200, first, second)
		}},
		{"vertical", `default="150" vertical`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 150, widget.Vertical(), first, second)
		}},
		{"ratio", `default="0.25" as-ratio`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 0.25, widget.AsRatio(), first, second)
		}},
		{"first minimum", `default="10" min-first="120"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 10, widget.MinFirst(120), first, second)
		}},
		{"second minimum", `default="590" min-second="150"`, func(state *widget.SplitState, first, second view.Node) view.Node {
			return widget.Split(state, 590, widget.MinSecond(150), first, second)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &widget.SplitState{}
			template := build(t, `<split state="split" `+test.attributes+`><row bg="panel"/><row bg="frame"/></split>`, Scope{"split": state})
			goView := test.goView(state, view.Row(view.Bg(core.Panel)), view.Row(view.Bg(core.Frame)))
			sameLayout(t, testCtx(t), template, goView, 601, 400)
		})
	}
}

func TestSplitElementDragMovesTheStateOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	state := &widget.SplitState{}
	template := build(t, `<split state="split" default="200"><row/><row/></split>`, Scope{"split": state})
	view.Layout(ctx, template, 600, 100)
	ctx.In = core.Input{X: 200, Y: 40, Click: true, Held: true}
	view.Update(ctx, template)
	ctx.In = core.Input{X: 320, Y: 40, Held: true}
	view.Update(ctx, template)
	if state.Size != 320 {
		t.Fatalf("after dragging the divider to x=320 the scope's state holds %v", state.Size)
	}
}

func TestTabsElementsMatchTheGoTabs(t *testing.T) {
	tabs := []widget.TabItem{{Label: "Body"}, {Label: "Headers", Badge: 3}, {Label: "Settings"}}
	strip := &widget.ScrollState{}
	choose := func(int) {}
	template := build(t, `
		<col>
		  <tabs items="tabs" active="1" on-select="choose" h="30"/>
		  <scroll-tabs state="strip" items="tabs" active="{{active}}" w="100"/>
		</col>`, Scope{"tabs": tabs, "strip": strip, "choose": choose, "active": 2})
	goView := view.Col(
		widget.Tabs(tabs, 1, view.OnSelect(choose), view.H(30)),
		widget.ScrollTabs(strip, tabs, 2, view.W(100)),
	)
	sameLayout(t, testCtx(t), template, goView, 400, 100)
}

func TestTabsElementClickSelectsThroughTheScopeHandler(t *testing.T) {
	ctx := testCtx(t)
	chosen := -1
	template := build(t, `<col><tabs items="tabs" active="0" on-select="choose"/></col>`,
		Scope{"tabs": []widget.TabItem{{Label: "One"}, {Label: "Two"}}, "choose": func(index int) { chosen = index }})
	view.Layout(ctx, template, 400, 100)
	second := template.Children()[0].Children()[1].Rect()
	ctx.In = core.Input{X: second.X + second.W/2, Y: second.Y + second.H/2, Click: true}
	view.Update(ctx, template)
	if chosen != 1 {
		t.Fatalf("clicking the second tab chose %d, want 1", chosen)
	}
}

func TestSeparatorAndActiveTabAttributesMatchTheGoOptions(t *testing.T) {
	got := view.Dump(build(t, `<row separator="12"><row active-tab/></row>`, nil))
	want := view.Dump(view.Row(view.Separator(12), view.Row(widget.ActiveTab())))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

// A zero separator switches it off, so a tab strip's template puts the
// separator on every tab but the first.
func TestZeroSeparatorAppliesNoOption(t *testing.T) {
	got := view.Dump(build(t, `<row separator="{{width}}"/>`, Scope{"width": 0}))
	if want := view.Dump(view.Row()); got != want {
		t.Fatalf("template:\n%s\nwant:\n%s", got, want)
	}
}

func TestTabsTakeNeitherChildrenNorText(t *testing.T) {
	for _, src := range []string{`<tabs items="x" active="0"><row/></tabs>`, `<tabs items="x" active="0">label</tabs>`} {
		if _, err := Parse("t.vml", []byte(src)); err == nil {
			t.Errorf("%s parsed", src)
		}
	}
}

// tableSpec is an editable two-column table whose edit requests land in edits.
func tableSpec(edits *[][2]int) widget.TableSpec {
	return widget.TableSpec{
		Columns:     []string{"Name", "Value"},
		Weights:     []float64{1, 2},
		Rows:        []widget.TableRow{{Cells: []string{"accept", "json"}, Checked: true}, {Cells: []string{"token", "abc"}, Muted: true}},
		Placeholder: []string{"Key", "Value"},
		Editable:    true,
		OnToggle:    func(int) {},
		OnDelete:    func(int) {},
		OnEditCell:  func(row, col int) { *edits = append(*edits, [2]int{row, col}) },
	}
}

func TestTableElementMatchesTheGoTable(t *testing.T) {
	var edits [][2]int
	spec := tableSpec(&edits)
	template := build(t, `<col><table spec="spec" margin-t="4"/></col>`, Scope{"spec": spec})
	goView := view.Col(widget.Table(spec, view.MarginT(4)))
	sameLayout(t, testCtx(t), template, goView, 400, 300)
}

func TestTableElementCellClickRunsTheSpecOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	var edits [][2]int
	template := build(t, `<col><table spec="spec"/></col>`, Scope{"spec": tableSpec(&edits)})
	view.Layout(ctx, template, 400, 300)
	table := template.Children()[0]
	secondRow := table.Children()[2].Rect() // after the header
	ctx.In = core.Input{X: secondRow.X + secondRow.W - 60, Y: secondRow.Y + secondRow.H/2, Click: true}
	view.Update(ctx, template)
	if len(edits) != 1 || edits[0] != [2]int{1, 1} {
		t.Fatalf("clicking the value of the second row asked to edit %v, want [[1 1]]", edits)
	}
}

func TestTableSpecOfTheWrongTypeNamesTheTypeItWants(t *testing.T) {
	var edits [][2]int
	spec := tableSpec(&edits)
	err := buildError(t, `<table spec="spec"/>`, Scope{"spec": &spec})
	if err == nil || !strings.Contains(err.Error(), "widget.TableSpec") {
		t.Errorf("error = %v, want it to say spec is not a widget.TableSpec", err)
	}
}

func TestDialogElementsMatchTheGoDialogs(t *testing.T) {
	closeDialog := func() {}
	template := build(t, `
		<stack>
		  <row grow="1"/>
		  <dialog w="300" h="200" on-dismiss="close">
		    <dialog-title title="Rename {{name}}" on-dismiss="close"><button>Help</button></dialog-title>
		    <text>body</text>
		  </dialog>
		</stack>`, Scope{"close": closeDialog, "name": "users"})
	goView := view.Stack(
		view.Row(view.Grow(1)),
		widget.Dialog(view.W(300), view.H(200), view.OnDismiss(closeDialog),
			widget.DialogTitle("Rename users", view.OnDismiss(closeDialog), widget.Button("Help")),
			view.Text("body"),
		),
	)
	sameLayout(t, testCtx(t), template, goView, 600, 400)
}

func TestMessageDialogElementMatchesTheGoMessageDialog(t *testing.T) {
	state := &widget.DialogState{Title: "Delete?", Body: "It cannot be undone.", Buttons: []widget.DialogButton{
		{Label: "Cancel", Run: func() {}}, {Label: "Delete", Primary: true, Run: func() {}},
	}}
	closeDialog := func() {}
	template := build(t, `<stack><row grow="1"/><message-dialog state="dialog" on-dismiss="close"><text small>extra</text></message-dialog></stack>`,
		Scope{"dialog": state, "close": closeDialog})
	goView := view.Stack(view.Row(view.Grow(1)), widget.MessageDialog(state, view.OnDismiss(closeDialog), view.Text("extra", view.Small())))
	sameLayout(t, testCtx(t), template, goView, 600, 400)
}

func TestDialogElementOutsideClickRunsTheScopeDismiss(t *testing.T) {
	ctx := testCtx(t)
	dismissed := 0
	template := build(t, `<stack><row grow="1"/><dialog w="200" h="100" on-dismiss="close"><text>hi</text></dialog></stack>`,
		Scope{"close": func() { dismissed++ }})
	view.Layout(ctx, template, 600, 400)
	ctx.In = core.Input{X: 5, Y: 5, Click: true}
	view.Update(ctx, template)
	if dismissed != 1 {
		t.Fatalf("an outside click dismissed %d times, want 1", dismissed)
	}
}

func TestFocusScopeAttributeMatchesTheGoOption(t *testing.T) {
	got := view.Dump(build(t, `<col focus-scope modal/>`, nil))
	want := view.Dump(view.Col(view.FocusScope(), view.Modal()))
	if got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func menuState() *widget.MenuState {
	return widget.NewMenu(20, 30, []widget.MenuItem{
		{Label: "Rename", Shortcut: "F2", Icon: "plus"}, {Separator: true}, {Label: "Locked", Disabled: true}, {Label: "Delete", Hint: "removes it"},
	})
}

func TestMenuElementMatchesTheGoMenu(t *testing.T) {
	ctx := testCtx(t)
	state := menuState()
	pick := func(int) {}
	template := build(t, `<stack><menu ctx="ctx" state="menu" on-select="pick"/></stack>`, Scope{"ctx": ctx, "menu": state, "pick": pick})
	goView := view.Stack(widget.Menu(ctx, state, view.OnSelect(pick)))
	sameLayout(t, ctx, template, goView, 600, 400)
}

func TestMenuElementClickPicksThroughTheScopeHandler(t *testing.T) {
	ctx := testCtx(t)
	var picked []int
	template := build(t, `<stack><menu ctx="ctx" state="menu" on-select="pick"/></stack>`,
		Scope{"ctx": ctx, "menu": menuState(), "pick": func(index int) { picked = append(picked, index) }})
	view.Layout(ctx, template, 600, 400)
	last := template.Children()[0].Children()[3].Rect()
	ctx.In = core.Input{X: last.X + last.W/2, Y: last.Y + last.H/2, Click: true}
	view.Update(ctx, template)
	if len(picked) != 1 || picked[0] != 3 {
		t.Fatalf("clicking the last row picked %v, want [3]", picked)
	}
}

func toastStack() *widget.ToastStack {
	stack := &widget.ToastStack{}
	stack.Show("saved", false, time.Unix(0, 0))
	stack.Show("boom", true, time.Unix(0, 0))
	return stack
}

func TestToastsElementMatchesTheGoToasts(t *testing.T) {
	stack := toastStack()
	template := build(t, `<stack><toasts stack="toasts" width="{{w}}" height="600" bottom-inset="28"/></stack>`,
		Scope{"toasts": stack, "w": 800})
	goView := view.Stack(widget.Toasts(stack, 800, 600, 28))
	sameLayout(t, testCtx(t), template, goView, 800, 600)
}

func TestToastsElementClickDismissesFromTheStackOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	stack := toastStack()
	template := build(t, `<stack><toasts stack="toasts" width="800" height="600" bottom-inset="28"/></stack>`, Scope{"toasts": stack})
	view.Layout(ctx, template, 800, 600)
	first := template.Children()[0].Children()[0].Rect()
	ctx.In = core.Input{X: first.X + 5, Y: first.Y + 5, Click: true}
	view.Update(ctx, template)
	if len(*stack) != 1 || (*stack)[0].Text != "boom" {
		t.Fatalf("clicking the first toast left %+v", *stack)
	}
}

func TestToastsElementOfAnEmptyStackBuildsNothing(t *testing.T) {
	node := build(t, `<toasts stack="toasts" width="800" height="600" bottom-inset="28"/>`, Scope{"toasts": &widget.ToastStack{}})
	if node != nil {
		t.Fatalf("an empty stack built\n%s", view.Dump(node))
	}
}

// widget.Toasts places itself from the window size and takes no options,
// so an option on it would have nothing to apply to.
func TestToastsElementTakesNoOptions(t *testing.T) {
	err := buildError(t, `<toasts stack="toasts" width="800" height="600" bottom-inset="28" bg="panel"/>`, Scope{"toasts": toastStack()})
	if err == nil || !strings.Contains(err.Error(), "no options") {
		t.Errorf("error = %v, want it to say <toasts> takes no options", err)
	}
}

// Split's options set the split behavior, so another element must not take them.
func TestSplitOptionsAreOnlyForSplit(t *testing.T) {
	for _, attribute := range []string{"vertical", "as-ratio", `min-first="1"`, `min-second="1"`, `default="1"`} {
		if _, err := Parse("t.vml", []byte(`<row `+attribute+`/>`)); err == nil {
			t.Errorf("<row %s> parsed", attribute)
		}
	}
}
