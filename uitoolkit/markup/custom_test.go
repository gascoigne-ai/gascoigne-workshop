package markup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// knobWidth and knobHeight are the size a knob asks for.
const (
	knobWidth  = 60
	knobHeight = 20
)

// knob is a component with its own measure, layout and input, the way a
// view.Custom behavior of the app has: it asks for a fixed size, places
// its one child in its right half and counts the clicks it gets.
type knob struct {
	view.Base
	presses *int
}

func (knob) Measure(*view.El, *core.Ctx, int) (int, int) { return knobWidth, knobHeight }

func (knob) Layout(element *view.El, ctx *core.Ctx) {
	area := element.Rect()
	half := area.W / 2
	element.Children()[0].Place(ctx, core.Rect{X: area.X + half, Y: area.Y, W: area.W - half, H: area.H})
}

func (behavior knob) Update(element *view.El, ctx *core.Ctx) {
	if ctx.In.Click && element.Rect().Contains(ctx.In.X, ctx.In.Y) {
		*behavior.presses++
	}
}

func (behavior knob) Describe() string { return fmt.Sprintf("presses=%d", *behavior.presses) }

// newKnob is a knob element counting into presses, with a text child.
func newKnob(presses *int) *view.El {
	return view.Custom(knob{presses: presses}, view.Text("on"))
}

// clickKnob clicks the middle of the knob's right half in a laid-out tree.
func clickKnob(ctx *core.Ctx, root view.Node, knobElement *view.El) {
	area := knobElement.Rect()
	ctx.In = core.Input{X: area.X + area.W*3/4, Y: area.Y + area.H/2, Click: true}
	view.Update(ctx, root)
}

// A Custom node put in through a slot measures, lays out its child and
// takes its clicks as it does in a Go-built tree.
func TestSlotKeepsACustomNodesLayoutAndInput(t *testing.T) {
	ctx := testCtx(t)
	var templatePresses, goPresses int
	templateKnob, goKnob := newKnob(&templatePresses), newKnob(&goPresses)
	template := build(t, `<row h="30" gap="8"><text>volume</text><slot name="knob"/></row>`, Scope{"knob": templateKnob})
	goView := view.Row(view.H(30), view.Gap(8), view.Text("volume"), goKnob)
	sameLayout(t, ctx, template, goView, 300, 30)
	clickKnob(ctx, template, templateKnob)
	clickKnob(ctx, goView, goKnob)
	if templatePresses != 1 || goPresses != 1 {
		t.Fatalf("a click on the knob counted %d in the template and %d in Go, want 1 and 1", templatePresses, goPresses)
	}
	if got, want := view.Dump(template), view.Dump(goView); got != want {
		t.Fatalf("after the click, template:\n%s\nGo:\n%s", got, want)
	}
}

// A slot or include used to accept look and handler attributes at parse
// and then drop them at build, so `<slot name="x" grow="1"/>` built a node
// that did not grow and said nothing. They are a parse error now; name,
// if and for, which both honor, are still accepted.
func TestSlotAndIncludeRejectAttributesTheyWouldDrop(t *testing.T) {
	for _, tag := range []string{"slot", "include"} {
		for _, attribute := range []string{`grow="1"`, `bg="panel"`, `on-click="act"`} {
			_, err := Parse("t.vml", []byte(`<row><`+tag+` name="x" `+attribute+`/></row>`))
			if err == nil || !strings.Contains(err.Error(), "on <"+tag+">") {
				t.Errorf("<%s %s>: err = %v, want an unknown attribute on <%s>", tag, attribute, err, tag)
			}
		}
		for _, attribute := range []string{`if="show"`, `for="item in items"`} {
			if _, err := Parse("t.vml", []byte(`<row><`+tag+` name="x" `+attribute+`/></row>`)); err != nil {
				t.Errorf("<%s %s>: %v", tag, attribute, err)
			}
		}
	}
}

// knobRow is a list item that carries its own Custom node.
type knobRow struct {
	Label string
	Knob  *view.El
}

// A slot names a path, so each item of a for puts in its own Custom node,
// and each keeps its own input.
func TestSlotInAForPutsInEachItemsCustomNode(t *testing.T) {
	ctx := testCtx(t)
	presses := make([]int, 2)
	goPresses := make([]int, 2)
	rows := []knobRow{{"bass", newKnob(&presses[0])}, {"treble", newKnob(&presses[1])}}
	goRows := []knobRow{{"bass", newKnob(&goPresses[0])}, {"treble", newKnob(&goPresses[1])}}
	template := build(t, `
		<col>
		  <row for="row in rows" h="30"><text>{{row.Label}}</text><slot name="row.Knob"/></row>
		</col>`, Scope{"rows": rows})
	goView := view.Col(view.For(goRows, func(_ int, row knobRow) view.Node {
		return view.Row(view.H(30), view.Text(row.Label), row.Knob)
	}))
	sameLayout(t, ctx, template, goView, 300, 60)
	clickKnob(ctx, template, rows[1].Knob)
	if presses[0] != 0 || presses[1] != 1 {
		t.Fatalf("a click on the second knob counted %v, want [0 1]", presses)
	}
}
