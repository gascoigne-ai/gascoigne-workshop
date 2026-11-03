package ui

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
)

// CodeView is a read-only, scrollable, foldable text viewer with line
// numbers.
type CodeView struct {
	Lines   []jsonview.Line
	Folded  map[int]bool // by original line index
	ScrollY int          // pixels
}

// SetBody replaces the content and resets folds and scroll.
func (v *CodeView) SetBody(body []byte) {
	v.Lines = jsonview.Lines(body)
	v.Folded = map[int]bool{}
	v.ScrollY = 0
}

// ToggleFold folds or unfolds the foldable line at original index i.
func (v *CodeView) ToggleFold(i int) {
	if i < 0 || i >= len(v.Lines) || !v.Lines[i].Foldable {
		return
	}
	if v.Folded == nil {
		v.Folded = map[int]bool{}
	}
	v.Folded[i] = !v.Folded[i]
}

// Visible lists the original indexes of the lines to show. A folded
// line hides everything up to and including its closing line.
func (v *CodeView) Visible() []int {
	var out []int
	for i := 0; i < len(v.Lines); i++ {
		out = append(out, i)
		if !v.Folded[i] || !v.Lines[i].Foldable {
			continue
		}
		indent := indentOf(v.Lines[i])
		for i+1 < len(v.Lines) {
			i++
			if indentOf(v.Lines[i]) <= indent {
				break
			}
		}
	}
	return out
}

// indentOf counts the leading spaces of a line.
func indentOf(l jsonview.Line) int {
	if len(l.Spans) == 0 {
		return 0
	}
	first := l.Spans[0]
	if first.Kind != jsonview.Plain {
		return 0
	}
	return len(first.Text) - len(strings.TrimLeft(first.Text, " "))
}

// Scroll moves the view by dy pixels, keeping it inside the content of
// contentH pixels shown in a viewport of viewH.
func (v *CodeView) Scroll(dy, contentH, viewH int) {
	v.ScrollY = min(max(v.ScrollY+dy, 0), max(contentH-viewH, 0))
}
