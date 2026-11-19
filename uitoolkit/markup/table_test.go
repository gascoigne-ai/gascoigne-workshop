package markup

import (
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Characterization of the exact Parse and Build error texts that the table
// refactor must keep.
func TestParseErrorTexts(t *testing.T) {
	for name, tt := range map[string]struct{ src, want string }{
		"unknown element":   {"<row>\n<blink/></row>", `t.vml:2: unknown element <blink>`},
		"near element":      {"<rows/>", `(did you mean "row"?)`},
		"unknown attribute": {"<text sparkle=\"1\">x</text>", `t.vml:1: unknown attribute "sparkle" on <text>`},
		"missing param":     {"<vrule-inset/>", "<vrule-inset> needs a inset attribute"},
		"cannot contain":    {"<text><row/></text>", "<text> cannot contain <row>"},
		"two roots":         {"<row/><row/>", "a template has one root element"},
		"text outside":      {"loose", "text outside the root element"},
		"text in row":       {"<row>loose</row>", "text is only allowed inside <text>, <caption> and <button>, not <row>"},
		"stand-in option":   {`<slot name="x" grow="1"/>`, `unknown attribute "grow" on <slot>: it takes only name, if and for`},
		"name on a row":     {`<row name="x"/>`, `unknown attribute "name" on <row>`},
	} {
		_, err := Parse("t.vml", []byte(tt.src))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestUnknownElementListsEveryElementOfTheTable(t *testing.T) {
	_, err := Parse("t.vml", []byte("<blink/>"))
	if err == nil {
		t.Fatal("parsed")
	}
	_, list, found := strings.Cut(err.Error(), "elements: ")
	if !found {
		t.Fatalf("no element list in %q", err)
	}
	if got, want := strings.Fields(list), elementNames(); !slices.Equal(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestSlotAndIconBuildErrors(t *testing.T) {
	for name, tt := range map[string]struct {
		src   string
		scope Scope
		want  string
	}{
		"slot is not a node":  {`<slot name="x"/>`, Scope{"x": "text"}, "is not a"},
		"slot is unknown":     {`<slot name="x"/>`, Scope{}, "unknown name"},
		"icon is unknown":     {`<icon name="nope"/>`, nil, `no icon "nope"; icons: `},
		"icon from the scope": {`<icon name="{{which}}"/>`, Scope{"which": "nope"}, `no icon "nope"`},
	} {
		err := buildError(t, tt.src, tt.scope)
		if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.HasPrefix(err.Error(), "t.vml:1: ") {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestSlotAndIconNeedAName(t *testing.T) {
	for _, src := range []string{`<slot/>`, `<icon/>`} {
		_, errParse := Parse("t.vml", []byte(src))
		if errParse != nil {
			continue
		}
		if err := buildError(t, src, nil); err == nil {
			t.Errorf("%s built without a name", src)
		}
	}
}

func TestSlotInAnIfIsNotLookedUpWhenHidden(t *testing.T) {
	node := build(t, `<row><slot name="missing" if="show"/></row>`, Scope{"show": false})
	if got, want := view.Dump(node), view.Dump(view.Row()); got != want {
		t.Fatalf("hidden slot:\n%s\nwant:\n%s", got, want)
	}
}

// <spacer>abc</spacer> parsed and dropped the text: an element that is built
// from options alone has nothing to put it in, so text is a parse error.
func TestElementsWithoutTextRejectText(t *testing.T) {
	for _, src := range []string{
		`<spacer>abc</spacer>`, `<hrule>abc</hrule>`, `<vrule>abc</vrule>`,
		`<vrule-inset inset="4">abc</vrule-inset>`, `<slot name="x">abc</slot>`,
	} {
		_, err := Parse("t.vml", []byte(src))
		if err == nil || !strings.Contains(err.Error(), "text is only allowed") {
			t.Errorf("%s: err = %v, want a text error", src, err)
		}
	}
}
