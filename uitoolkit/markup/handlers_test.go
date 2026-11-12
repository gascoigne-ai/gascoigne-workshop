package markup

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Each handler attribute is the Go option in kebab-case and takes a func of
// the scope; the template and the Go view dump the same, handlers included.
func TestHandlerAttributesMatchTheGoOptions(t *testing.T) {
	tests := []struct {
		attribute string
		handler   any
		option    view.Option
	}{
		{"on-click", func() {}, view.OnClick(func() {})},
		{"on-click-at", func(int, int) {}, view.OnClickAt(func(int, int) {})},
		{"on-right-click", func(int, int) {}, view.OnRightClick(func(int, int) {})},
		{"on-right-click-in", func(*core.Ctx, int, int) {}, view.OnRightClickIn(func(*core.Ctx, int, int) {})},
		{"on-middle-click", func() {}, view.OnMiddleClick(func() {})},
		{"on-drag-over", func() {}, view.OnDragOver(func() {})},
		{"on-change", func() {}, view.OnChange(func() {})},
		{"on-select", func(int) {}, view.OnSelect(func(int) {})},
		{"on-dismiss", func() {}, view.OnDismiss(func() {})},
	}
	for _, test := range tests {
		t.Run(test.attribute, func(t *testing.T) {
			node := build(t, `<row `+test.attribute+`="act"/>`, Scope{"act": test.handler})
			if got, want := view.Dump(node), view.Dump(view.Row(test.option)); got != want {
				t.Errorf("template dumps %q, Go view dumps %q", got, want)
			}
		})
	}
}

func TestHandlerAttributeCallsTheScopeFunction(t *testing.T) {
	var gotX, gotY int
	node := build(t, `<row on-right-click="menu"/>`, Scope{"menu": func(x, y int) { gotX, gotY = x, y }})
	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	view.Layout(c, node, 100, 100)
	c.In = core.Input{X: 7, Y: 9, RightClick: true}
	view.Update(c, node)
	if gotX != 7 || gotY != 9 {
		t.Errorf("right click handler got (%d,%d), want (7,9)", gotX, gotY)
	}
}

func TestHandlerAttributeOfTheWrongTypeNamesTheTypeItWants(t *testing.T) {
	template, err := Parse("t.vml", []byte(`<row on-right-click="act"/>`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = template.Build(Scope{"act": func() {}})
	if err == nil || !strings.Contains(err.Error(), "func(int, int)") {
		t.Errorf("error = %v, want it to say act is not a func(int, int)", err)
	}
}
