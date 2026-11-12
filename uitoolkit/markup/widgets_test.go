package markup

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"

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

func TestImageElementMatchesTheGoImage(t *testing.T) {
	picture := ebiten.NewImage(80, 40)
	template := build(t, `<row><image img="picture" grow="1" bg="sunken" tooltip="The image"/></row>`, Scope{"picture": picture})
	goView := view.Row(widget.Image(picture, view.Grow(1), view.Bg(core.Sunken), view.Tooltip("The image")))
	sameWidget(t, template, goView, 400, 100)
}

func TestImageElementMeasuresThePictureOfTheScope(t *testing.T) {
	template := build(t, `<row align-items="center"><image img="picture"/></row>`, Scope{"picture": ebiten.NewImage(80, 40)})
	view.Layout(testCtx(t), template, 200, 100)
	if size := template.Children()[0].Rect(); size.W != 80 || size.H != 40 {
		t.Fatalf("the image is %dx%d, want the 80x40 of the scope's picture", size.W, size.H)
	}
}

// widget.Image draws nothing for a nil picture (one not decoded yet), so a
// nil *ebiten.Image is not the mistake a nil state is.
func TestImageElementTakesANilPicture(t *testing.T) {
	if err := buildError(t, `<image img="picture"/>`, Scope{"picture": (*ebiten.Image)(nil)}); err != nil {
		t.Fatal(err)
	}
	err := buildError(t, `<image img="picture"/>`, Scope{"picture": "cat.png"})
	if err == nil || !strings.Contains(err.Error(), "*ebiten.Image") {
		t.Errorf("error = %v, want it to say picture is not a *ebiten.Image", err)
	}
}

func TestSliderElementMatchesTheGoSlider(t *testing.T) {
	state := &widget.SliderState{}
	slide, end := func(int) {}, func(int) {}
	template := build(t, `<row><slider state="scale" min="50" max="200" step="5" value="{{percent}}" slider-mid="100" on-slide="slide" on-slide-end="end" w="214"/></row>`,
		Scope{"scale": state, "percent": 125, "slide": slide, "end": end})
	goView := view.Row(widget.Slider(state, 50, 200, 5, 125, widget.SliderMid(100), widget.OnSlide(slide), widget.OnSlideEnd(end), view.W(214)))
	sameWidget(t, template, goView, 400, 40)
}

// The track of a w="214" slider is 200px from x=7: half a thumb of room
// at each end.
const sliderTrackStart = 7

func TestSliderElementDragMovesTheStateAndRunsTheScopeHandlers(t *testing.T) {
	ctx := testCtx(t)
	state := &widget.SliderState{}
	var live, ended []int
	template := build(t, `<row><slider state="scale" min="50" max="200" step="5" value="100" slider-mid="100" on-slide="slide" on-slide-end="end" w="214"/></row>`,
		Scope{"scale": state, "slide": func(value int) { live = append(live, value) }, "end": func(value int) { ended = append(ended, value) }})
	view.Layout(ctx, template, 400, 40)
	ctx.In = core.Input{X: sliderTrackStart + 50, Y: 20, Click: true, Held: true} // a quarter: halfway from 50 to the mid 100
	view.Update(ctx, template)
	if !state.Dragging || len(live) != 1 || live[0] != 75 {
		t.Fatalf("the press left the scope's state dragging=%v and slid %v, want true and [75]", state.Dragging, live)
	}
	ctx.In = core.Input{X: sliderTrackStart + 50, Y: 20}
	view.Update(ctx, template)
	if state.Dragging || len(ended) != 1 || ended[0] != 75 {
		t.Fatalf("the release left dragging=%v and ended %v, want false and [75]", state.Dragging, ended)
	}
}

func TestSliderElementReadsItsRangeAsWholeNumbers(t *testing.T) {
	parseError(t, `<slider state="s" min="0" max="10" step="1"/>`, "<slider> needs a value")
	err := buildError(t, `<slider state="s" min="0" max="ten" step="1" value="0"/>`, Scope{"s": &widget.SliderState{}})
	if err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("error = %v, want it to say max is not a whole number", err)
	}
}

// The slider's options set the slider behavior; another element ignored
// them in Go, so a template rejects them instead of dropping them.
func TestSliderOptionsAreOnlyForSlider(t *testing.T) {
	for _, attribute := range []string{`slider-mid="1"`, `on-slide="f"`, `on-slide-end="f"`} {
		parseError(t, `<row `+attribute+`/>`, "unknown attribute")
	}
}

func monoFace(t *testing.T) *text.GoTextFaceSource {
	t.Helper()
	face, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		t.Fatal(err)
	}
	return face
}

func TestFontSourceAttributeMatchesTheGoOption(t *testing.T) {
	face := monoFace(t)
	template := build(t, `<row><text font-source="face">iiii</text></row>`, Scope{"face": face})
	goView := view.Row(view.Text("iiii", view.FontSource(face)))
	sameLayout(t, testCtx(t), template, goView, 400, 30)
}

func TestFontSourceAttributeMeasuresInTheFaceOfTheScope(t *testing.T) {
	ctx := testCtx(t)
	template := build(t, `<row><text font-source="face">iiii</text><text>iiii</text></row>`, Scope{"face": monoFace(t)})
	view.Layout(ctx, template, 400, 30)
	inFace, inInterface := template.Children()[0].Rect().W, template.Children()[1].Rect().W
	if inFace <= inInterface {
		t.Fatalf("iiii is %dpx wide in the monospaced face and %dpx in the interface font; want it wider in the face", inFace, inInterface)
	}
}

// A font list keeps a nil face for a file it could not read, and Go draws
// that text in the interface font.
func TestFontSourceAttributeTakesANilFace(t *testing.T) {
	if err := buildError(t, `<text font-source="face">x</text>`, Scope{"face": (*text.GoTextFaceSource)(nil)}); err != nil {
		t.Fatal(err)
	}
	err := buildError(t, `<text font-source="face">x</text>`, Scope{"face": "Go Mono"})
	if err == nil || !strings.Contains(err.Error(), "*text.GoTextFaceSource") {
		t.Errorf("error = %v, want it to say face is not a *text.GoTextFaceSource", err)
	}
}

// resizableDialogTemplate is a 300x200 dialog, between 120x80 and 500x400,
// centered in an 800x600 window: from x=250 to x=550.
const resizableDialogTemplate = `
	<stack>
	  <row grow="1"/>
	  <resizable-dialog state="size" min-width="120" min-height="80" max-width="{{maxWidth}}" max-height="400" w="300" h="200" on-dismiss="close">
	    <text>body</text>
	  </resizable-dialog>
	</stack>`

func TestResizableDialogElementMatchesTheGoResizableDialog(t *testing.T) {
	size := &widget.ResizeState{}
	closeDialog := func() {}
	template := build(t, resizableDialogTemplate, Scope{"size": size, "maxWidth": 500, "close": closeDialog})
	goView := view.Stack(view.Row(view.Grow(1)),
		widget.ResizableDialog(size, 120, 80, 500, 400, view.W(300), view.H(200), view.OnDismiss(closeDialog), view.Text("body")))
	sameLayout(t, testCtx(t), template, goView, 800, 600)
	last := func(node view.Node) string {
		children := node.Children()
		return fmt.Sprintf("%+v", children[len(children)-1].Behavior())
	}
	if got, want := last(template), last(goView); got != want {
		t.Fatalf("template dialog:\n%s\nGo dialog:\n%s", got, want)
	}
}

func TestResizableDialogElementDragResizesTheStateWithinItsBounds(t *testing.T) {
	ctx := testCtx(t)
	size := &widget.ResizeState{}
	template := build(t, resizableDialogTemplate, Scope{"size": size, "maxWidth": 500, "close": func() {}})
	view.Layout(ctx, template, 800, 600)
	for _, input := range []core.Input{{X: 548, Y: 300, Click: true, Held: true}, {X: 2000, Y: 300, Held: true}} {
		ctx.In = input
		view.Update(ctx, template)
	}
	if size.W != 500 {
		t.Fatalf("dragging the right edge far out left the scope's state %d wide, want max-width 500", size.W)
	}
	ctx.In = core.Input{X: -2000, Y: 300, Held: true}
	view.Update(ctx, template)
	if size.W != 120 {
		t.Fatalf("dragging the right edge far in left the scope's state %d wide, want min-width 120", size.W)
	}
}

func TestResizableDialogNeedsItsStateAndFourBounds(t *testing.T) {
	parseError(t, `<resizable-dialog state="s" min-width="1" min-height="1" max-width="9"/>`, "<resizable-dialog> needs a max-height")
	err := buildError(t, `<resizable-dialog state="s" min-width="1" min-height="1" max-width="9" max-height="9"/>`, Scope{"s": (*widget.ResizeState)(nil)})
	if err == nil || !strings.Contains(err.Error(), "nil *widget.ResizeState") {
		t.Errorf("error = %v, want it to say s is a nil *widget.ResizeState", err)
	}
}

func TestCheckboxOnMustBeTrueOrFalse(t *testing.T) {
	parseError(t, `<checkbox/>`, "<checkbox> needs a on")
	err := buildError(t, `<checkbox on="yes"/>`, nil)
	if err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Errorf("error = %v, want it to say on is not true or false", err)
	}
}
