package respview

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
)

func lineText(l widget.Line) string {
	var sb strings.Builder
	for _, s := range l.Spans {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// blockEnds is, for every foldable line, the index of the line that
// closes its block: the next line that starts with } or ] after the
// blocks opened inside it have been closed.
func blockEnds(lines []widget.Line) map[int]int {
	ends := map[int]int{}
	var stack []int
	for i, l := range lines {
		if l.Foldable {
			stack = append(stack, i)
			continue
		}
		if t := strings.TrimSpace(lineText(l)); (strings.HasPrefix(t, "}") || strings.HasPrefix(t, "]")) && len(stack) > 0 {
			ends[stack[len(stack)-1]] = i
			stack = stack[:len(stack)-1]
		}
	}
	return ends
}

// Filter keeps only what a search needs from the compact lines:
// the lines that contain query (any case), all lines inside a matching
// block (its children), and the opening and closing lines of the blocks
// around them, so every match is shown in its place in the tree.
func Filter(lines []widget.Line, query string) []widget.Line {
	q := strings.ToLower(query)
	n := len(lines)
	ends := blockEnds(lines)
	keep := make([]bool, n)
	for i := 0; i < n; i++ {
		if !strings.Contains(strings.ToLower(lineText(lines[i])), q) {
			continue
		}
		keep[i] = true
		if end, ok := ends[i]; ok && lines[i].Foldable {
			for j := i; j <= end && j < n; j++ {
				keep[j] = true
			}
			i = end // everything inside is kept already
		}
	}
	// The blocks around kept lines stay open: keep their opener and closer.
	type open struct{ line, end int }
	var stack []open
	enclosing := make([]bool, n)
	for i := 0; i < n; i++ {
		for len(stack) > 0 && i > stack[len(stack)-1].end {
			stack = stack[:len(stack)-1]
		}
		if keep[i] {
			for _, s := range stack {
				enclosing[s.line], enclosing[s.end] = true, true
			}
		}
		if end, ok := ends[i]; ok && lines[i].Foldable {
			stack = append(stack, open{i, end})
		}
	}
	var out []widget.Line
	for i := 0; i < n; i++ {
		if keep[i] || enclosing[i] {
			l := lines[i]
			out = append(out, l)
		}
	}
	return out
}
