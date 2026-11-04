package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// App-specific color tokens, added to the widget theme.
const methodTokenPrefix = "method."

// methodToken is the color token of an HTTP method; unknown methods
// resolve to the text color through Theme.Color's fallback.
func methodToken(method string) widget.Token {
	if method == "" {
		return widget.Muted
	}
	return widget.Token(methodTokenPrefix + method)
}

// jsonTokens colors the spans of a JSON response.
var jsonTokens = map[jsonview.Kind]widget.Token{
	jsonview.Plain:   "json.plain",
	jsonview.Punct:   "json.punct",
	jsonview.Key:     "json.key",
	jsonview.String:  "json.string",
	jsonview.Number:  "json.number",
	jsonview.Literal: "json.literal",
}

func appColors(methods map[string]uint32, key, str, num, lit uint32) map[widget.Token]color.RGBA {
	colors := map[widget.Token]color.RGBA{
		jsonTokens[jsonview.Key]:     widget.RGB(key),
		jsonTokens[jsonview.String]:  widget.RGB(str),
		jsonTokens[jsonview.Number]:  widget.RGB(num),
		jsonTokens[jsonview.Literal]: widget.RGB(lit),
	}
	for m, hex := range methods {
		colors[methodToken(m)] = widget.RGB(hex)
	}
	return colors
}

func darkTheme() widget.Theme {
	th := widget.Dark()
	extra := appColors(map[string]uint32{
		"GET": 0x5ed39a, "POST": 0xf0b05a, "PUT": 0x7db4ea, "PATCH": 0xe88aa6, "DELETE": 0xff6b5e,
	}, 0xffffff, 0xf0b05a, 0xe88aa6, 0xff6b5e)
	extra[jsonTokens[jsonview.Plain]] = th.Color(widget.Text)
	extra[jsonTokens[jsonview.Punct]] = th.Color(widget.Text)
	return th.With(extra)
}

func lightTheme() widget.Theme {
	th := widget.Light()
	extra := appColors(map[string]uint32{
		"GET": 0x17804a, "POST": 0xa8570c, "PUT": 0x1f64b0, "PATCH": 0x7a1f3d, "DELETE": 0xd1281b,
	}, 0x120a0e, 0xa8570c, 0x7a1f3d, 0xd1281b)
	extra[jsonTokens[jsonview.Plain]] = th.Color(widget.Text)
	extra[jsonTokens[jsonview.Punct]] = th.Color(widget.Text)
	return th.With(extra)
}

// themeChoice and sizeChoice are the entries of the settings menu.
type themeChoice struct {
	id, label string
	build     func() widget.Theme
}

type sizeChoice struct {
	id, label         string
	font, small, mono float64
}

var themeChoices = []themeChoice{
	{"dark", "Dark", darkTheme},
	{"light", "Light", lightTheme},
}

var textSizeChoices = []sizeChoice{
	{"small", "Small", 13, 11, 12},
	{"medium", "Medium", 15, 12, 14},
	{"large", "Large", 17, 13, 16},
}

// The interface scale slider's range and step, in percent.
const (
	scaleMin  = 50
	scaleMax  = 200
	scaleStep = 5
)

// ThemeFor builds the theme for a settings pair. Unknown ids fall back
// to the first (dark) theme and the medium text size.
func ThemeFor(themeID, textSizeID string) widget.Theme {
	th := themeChoices[0].build()
	for _, c := range themeChoices {
		if c.id == themeID {
			th = c.build()
		}
	}
	size := textSizeChoices[1]
	for _, c := range textSizeChoices {
		if c.id == textSizeID {
			size = c
		}
	}
	return th.WithTextSize(size.font, size.small, size.mono)
}

// codeLines turns a response body into colored code-view lines.
func codeLines(body []byte) []widget.Line {
	src := jsonview.Lines(body)
	lines := make([]widget.Line, len(src))
	for i, l := range src {
		spans := make([]widget.Span, len(l.Spans))
		for j, s := range l.Spans {
			spans[j] = widget.Span{Text: s.Text, Style: jsonTokens[s.Kind]}
		}
		lines[i] = widget.Line{Spans: spans, Foldable: l.Foldable}
	}
	return lines
}

// kvRows turns key/value rows into table rows with two or three cells.
func kvRows(rows []model.KV, withDesc, flagDuplicates bool) []vml.TableRow {
	out := make([]vml.TableRow, len(rows))
	var dup []bool
	if flagDuplicates {
		dup = model.DuplicateNames(rows)
	}
	for i, kv := range rows {
		cells := []string{kv.Name, kv.Value}
		if withDesc {
			cells = append(cells, kv.Description)
		}
		out[i] = vml.TableRow{Cells: cells, Checked: kv.Enabled, Muted: !kv.Enabled, Flagged: dup != nil && dup[i]}
	}
	return out
}

// layoutChoice is an entry of the layout chooser in the settings menu.
type layoutChoice struct{ id, label string }

var layoutChoices = []layoutChoice{
	{"side", "Side by side"},
	{"stacked", "Stacked"},
}
