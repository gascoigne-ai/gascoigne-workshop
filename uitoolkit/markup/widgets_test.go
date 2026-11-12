package markup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// sameWidget fails unless the template and the Go view dump the same once
// laid out and their first children, the widgets under test, were made
// with the same behavior. Dump shows only what an El holds; the widget's
// own settings (a placeholder, a slider's range, a checkbox's value) live
// in its behavior, which %+v prints field by field, funcs and state by
// address, so the scope's values must be the ones bound.
func sameWidget(t *testing.T, template, goView view.Node, width, height int) {
	t.Helper()
	sameLayout(t, testCtx(t), template, goView, width, height)
	got := fmt.Sprintf("%+v", template.Children()[0].Behavior())
	want := fmt.Sprintf("%+v", goView.Children()[0].Behavior())
	if got != want {
		t.Fatalf("template widget:\n%s\nGo widget:\n%s", got, want)
	}
}

// parseError fails unless src is a parse error that mentions every part.
func parseError(t *testing.T, src string, parts ...string) {
	t.Helper()
	_, err := Parse("t.vml", []byte(src))
	if err == nil {
		t.Fatalf("%s parsed", src)
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("%s: error %q does not mention %q", src, err, part)
		}
	}
}

func TestFieldElementMatchesTheGoField(t *testing.T) {
	url := widget.NewTextField("https://{{host}}/")
	paint := func(text string) []widget.Span { return []widget.Span{{Text: text, Style: core.Accent}} }
	edited, send, tab := func() {}, func() {}, func(bool) {}
	template := build(t, `
		<row>
		  <field state="url" grow="1" bg="sunken" placeholder="Enter URL" field-pad="5"
		    segments="paint" text-menu="url" on-change="edited" on-enter="send" on-tab="tab"/>
		</row>`, Scope{"url": &url, "paint": paint, "edited": edited, "send": send, "tab": tab})
	goView := view.Row(widget.Field(&url, view.Grow(1), view.Bg(core.Sunken), widget.Placeholder("Enter URL"), widget.FieldPad(5),
		widget.Segments(paint), widget.TextMenu(&url), view.OnChange(edited), widget.OnEnter(send), widget.OnTab(tab)))
	sameWidget(t, template, goView, 400, 30)
}

// focusedFieldTemplate is a 300px field, focused, editing the scope's
// TextField, with the scope's handlers.
func focusedFieldTemplate(t *testing.T, scope Scope) (*widget.TextField, *core.Ctx, view.Node) {
	t.Helper()
	field := widget.NewTextField("")
	field.Focused = true
	scope["field"] = &field
	template := build(t, `<row><field state="field" grow="1" on-enter="send" on-tab="tab"/></row>`, scope)
	ctx := testCtx(t)
	view.Layout(ctx, template, 300, 30)
	return &field, ctx, template
}

func TestFieldElementTypesIntoTheTextFieldOfTheScope(t *testing.T) {
	field, ctx, template := focusedFieldTemplate(t, Scope{"send": func() {}, "tab": func(bool) {}})
	ctx.In = core.Input{Typed: []rune("abc")}
	view.Update(ctx, template)
	if field.Text != "abc" {
		t.Fatalf("typing into the field left the scope's TextField at %q", field.Text)
	}
}

func TestFieldElementEnterAndTabRunTheScopeHandlers(t *testing.T) {
	sends, tabs := 0, []bool{}
	field, ctx, template := focusedFieldTemplate(t, Scope{"send": func() { sends++ }, "tab": func(back bool) { tabs = append(tabs, back) }})
	ctx.In = core.Input{Enter: true}
	view.Update(ctx, template)
	ctx.In = core.Input{Tab: true, Shift: true}
	view.Update(ctx, template)
	if sends != 1 || len(tabs) != 1 || !tabs[0] || !field.Focused {
		t.Fatalf("Enter sent %d times, Shift+Tab gave %v, focused %v; want 1, [true], true", sends, tabs, field.Focused)
	}
}

func TestFieldElementTakesNeitherChildrenNorText(t *testing.T) {
	parseError(t, `<field state="f"><row/></field>`, "cannot contain")
	parseError(t, `<field state="f">label</field>`, "text")
}

func TestFieldWithoutStateIsAParseError(t *testing.T) {
	parseError(t, `<field placeholder="x"/>`, "<field> needs a state")
}

func TestFieldStateMustBeATextField(t *testing.T) {
	area := widget.NewTextArea("")
	err := buildError(t, `<field state="f"/>`, Scope{"f": &area})
	if err == nil || !strings.Contains(err.Error(), "*widget.TextField") {
		t.Errorf("error = %v, want it to say f is not a *widget.TextField", err)
	}
	err = buildError(t, `<field state="f"/>`, Scope{"f": (*widget.TextField)(nil)})
	if err == nil || !strings.Contains(err.Error(), "nil *widget.TextField") {
		t.Errorf("error = %v, want it to say f is a nil *widget.TextField", err)
	}
}

// The field's options reach into the field's behavior: placeholder and
// field-pad were generic attributes, so <row placeholder> parsed and then
// panicked at build.
func TestFieldOptionsAreOnlyForField(t *testing.T) {
	for _, attribute := range []string{`placeholder="x"`, `field-pad="4"`, `segments="f"`, `text-menu="f"`, `on-enter="f"`, `on-tab="f"`} {
		t.Run(attribute, func(t *testing.T) {
			parseError(t, `<row `+attribute+`/>`, "unknown attribute")
		})
	}
}

func TestAreaElementMatchesTheGoArea(t *testing.T) {
	body := widget.NewTextArea("{\n  \"a\": 1\n}")
	edited := func() {}
	template := build(t, `<row><area state="body" grow="1" bg="sunken" on-change="edited" text-menu="bodyText"/></row>`,
		Scope{"body": &body, "bodyText": &body.TextField, "edited": edited})
	goView := view.Row(widget.Area(&body, view.Grow(1), view.Bg(core.Sunken), view.OnChange(edited), widget.TextMenu(&body.TextField)))
	sameWidget(t, template, goView, 400, 100)
}

func TestAreaElementTypesIntoTheTextAreaOfTheScope(t *testing.T) {
	body := widget.NewTextArea("")
	body.Focused = true
	changes := 0
	template := build(t, `<row><area state="body" grow="1" on-change="edited"/></row>`, Scope{"body": &body, "edited": func() { changes++ }})
	ctx := testCtx(t)
	view.Layout(ctx, template, 400, 100)
	for _, input := range []core.Input{{Typed: []rune("hi")}, {Enter: true}, {Typed: []rune("yo")}} {
		ctx.In = input
		view.Update(ctx, template)
	}
	if body.Text != "hi\nyo" || changes != 3 {
		t.Fatalf("the scope's TextArea holds %q after %d changes, want \"hi\\nyo\" after 3", body.Text, changes)
	}
}

func TestAreaStateMustBeATextArea(t *testing.T) {
	field := widget.NewTextField("")
	err := buildError(t, `<area state="a"/>`, Scope{"a": &field})
	if err == nil || !strings.Contains(err.Error(), "*widget.TextArea") {
		t.Errorf("error = %v, want it to say a is not a *widget.TextArea", err)
	}
}

func TestAreaElementTakesNeitherChildrenNorFieldOptions(t *testing.T) {
	parseError(t, `<area state="a"><row/></area>`, "cannot contain")
	parseError(t, `<area state="a" on-enter="f"/>`, "unknown attribute")
}

func TestCheckboxElementMatchesTheGoCheckbox(t *testing.T) {
	toggle := func() {}
	for _, on := range []bool{true, false} {
		t.Run(fmt.Sprint(on), func(t *testing.T) {
			template := build(t, `<row><checkbox on="{{enabled}}" on-change="toggle" tooltip="Follow redirects"/></row>`,
				Scope{"enabled": on, "toggle": toggle})
			goView := view.Row(widget.Checkbox(on, view.OnChange(toggle), view.Tooltip("Follow redirects")))
			sameWidget(t, template, goView, 200, 30)
		})
	}
}

func TestCheckboxElementClickRunsTheScopeChange(t *testing.T) {
	ctx := testCtx(t)
	toggles := 0
	template := build(t, `<row><checkbox on="true" on-change="toggle"/></row>`, Scope{"toggle": func() { toggles++ }})
	view.Layout(ctx, template, 200, 30)
	box := template.Children()[0].Rect()
	ctx.In = core.Input{X: box.X + box.W/2, Y: box.Y + box.H/2, Click: true}
	view.Update(ctx, template)
	if toggles != 1 {
		t.Fatalf("clicking the box ran the scope's on-change %d times, want 1", toggles)
	}
}

func codeLines() *widget.CodeView {
	code := &widget.CodeView{}
	code.SetLines([]widget.Line{{Spans: []widget.Span{{Text: "one"}}}, {Spans: []widget.Span{{Text: "two"}}}})
	return code
}

func TestCodeElementMatchesTheGoCode(t *testing.T) {
	code := codeLines()
	menu := func(int, int) {}
	template := build(t, `<row><code state="code" grow="1" bg="sunken" on-right-click="menu"/></row>`, Scope{"code": code, "menu": menu})
	goView := view.Row(widget.Code(code, view.Grow(1), view.Bg(core.Sunken), view.OnRightClick(menu)))
	sameWidget(t, template, goView, 400, 100)
}

func TestCodeElementSelectsInTheCodeViewOfTheScope(t *testing.T) {
	code := codeLines()
	template := build(t, `<row><code state="code" grow="1"/></row>`, Scope{"code": code})
	ctx := testCtx(t)
	view.Layout(ctx, template, 400, 100)
	ctx.In = core.Input{X: 300, Y: 15, Click: true, Held: true, Clicks: 3} // a triple click on the first line
	view.Update(ctx, template)
	if code.SelectedText() != "one" || !code.Focused {
		t.Fatalf("the scope's CodeView selected %q, focused %v; want \"one\", true", code.SelectedText(), code.Focused)
	}
}

func TestCodeStateMustBeACodeView(t *testing.T) {
	err := buildError(t, `<code state="c"/>`, Scope{"c": widget.CodeView{}})
	if err == nil || !strings.Contains(err.Error(), "*widget.CodeView") {
		t.Errorf("error = %v, want it to say c is not a *widget.CodeView", err)
	}
	parseError(t, `<code state="c">text</code>`, "text")
}

func TestCheckboxOnMustBeTrueOrFalse(t *testing.T) {
	parseError(t, `<checkbox/>`, "<checkbox> needs a on")
	err := buildError(t, `<checkbox on="yes"/>`, nil)
	if err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Errorf("error = %v, want it to say on is not true or false", err)
	}
}
