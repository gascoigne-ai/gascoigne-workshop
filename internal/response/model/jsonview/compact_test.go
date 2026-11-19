package jsonview

import (
	"strings"
	"testing"
)

func texts(lines []Line) []string {
	var out []string
	for _, l := range lines {
		out = append(out, text(l))
	}
	return out
}

func TestCompactPutsSmallValuesOnOneLine(t *testing.T) {
	got := texts(Compact([]byte(`{"a":1,"b":[1,2,3],"c":{"d":null}}`), 80))
	if len(got) != 1 || got[0] != `{"a": 1, "b": [1, 2, 3], "c": {"d": null}}` {
		t.Fatalf("%q", got)
	}
}

func TestCompactBreaksOnlyWhatDoesNotFit(t *testing.T) {
	body := `{"id":7,"owner":{"login":"jonegil","id":7612},"tags":["alpha","beta","gamma","delta","epsilon","zeta"],"note":"x"}`
	got := texts(Compact([]byte(body), 50))
	want := []string{
		`{`,
		`  "id": 7,`,
		`  "owner": {"login": "jonegil", "id": 7612},`,
		`  "tags": [`,
		`    "alpha",`,
		`    "beta",`,
		`    "gamma",`,
		`    "delta",`,
		`    "epsilon",`,
		`    "zeta"`,
		`  ],`,
		`  "note": "x"`,
		`}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestCompactMarksBlocksFoldableWithAPreviewForWhenTheyAreFolded(t *testing.T) {
	lines := Compact([]byte(`{"items":[{"id":1,"name":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","extra":"bbbbbbbbbbbbbbbbbbbb"},{"id":2}]}`), 40)
	var open *Line
	for i := range lines {
		if strings.HasPrefix(strings.TrimSpace(text(lines[i])), `{`) && strings.HasSuffix(text(lines[i]), `{`) && lines[i].Depth == 2 {
			open = &lines[i]
			break
		}
	}
	if open == nil || !open.Foldable {
		t.Fatalf("no foldable object line at depth 2 in %q", texts(lines))
	}
	preview := text(Line{Spans: open.Folded})
	if !strings.HasPrefix(preview, `    {"id": 1, "name": "aaa`) || !strings.HasSuffix(preview, `…},`) {
		t.Fatalf("preview %q", preview)
	}
	if len(preview) > 100 {
		t.Fatalf("preview too long: %d", len(preview))
	}
}

func TestCompactDepthFollowsNesting(t *testing.T) {
	lines := Compact([]byte(`{"a":{"b":{"c":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28]}}}`), 20)
	var depths []int
	for _, l := range lines {
		depths = append(depths, l.Depth)
	}
	if depths[0] != 0 || depths[1] != 1 || depths[2] != 2 {
		t.Fatalf("depths %v for %q", depths, texts(lines))
	}
}

func TestCompactNonJSONIsPlainAndStringsKeepTheirPunctuation(t *testing.T) {
	plainLines := Compact([]byte("hello\nworld"), 80)
	if len(plainLines) != 2 || plainLines[0].Foldable {
		t.Fatalf("%q", texts(plainLines))
	}
	got := texts(Compact([]byte(`{"s":"a, b: {c}","u":"é\"q"}`), 80))
	if got[0] != `{"s": "a, b: {c}", "u": "é\"q"}` {
		t.Fatalf("%q", got)
	}
}

func TestCompactKeepsKeyOrderAndLargeNumbersAsWritten(t *testing.T) {
	got := texts(Compact([]byte(`{"z":1,"a":12345678901234567890,"m":1.50}`), 80))
	if got[0] != `{"z": 1, "a": 12345678901234567890, "m": 1.50}` {
		t.Fatalf("%q", got)
	}
}

// preview compared bytes with previewMax but cut by runes, so a block of
// multi-byte text (over 80 bytes, under 78 runes) was sliced past its rune
// slice: it panicked or came back padded with NUL runes.
func TestCompactPreviewOfMultiByteTextIsCutByRunes(t *testing.T) {
	body := `{"a":{"k":"` + strings.Repeat("日本語", 18) + `"}}`
	lines := Compact([]byte(body), 20)
	var open Line
	for _, l := range lines {
		if len(l.Folded) > 0 {
			open = l
			break
		}
	}
	preview := text(Line{Spans: open.Folded})
	if preview == "" || strings.ContainsRune(preview, 0) {
		t.Fatalf("preview %q has no text or is padded with NUL runes", preview)
	}
	if !strings.HasSuffix(preview, "}") {
		t.Errorf("preview %q does not close the block", preview)
	}
}

// A body of thousands of nested brackets is valid JSON, and formatting it
// wrote one line per level, each indented by its depth: quadratic text
// built on the UI thread from a few KB of input. Past maxDepth the body
// is shown as it came.
func TestDeeplyNestedJSONIsShownAsPlainText(t *testing.T) {
	const depth = 3000
	body := []byte(strings.Repeat("[", depth) + strings.Repeat("]", depth))
	for name, lines := range map[string][]Line{"Compact": Compact(body, 80), "Lines": Lines(body)} {
		if len(lines) != 1 || text(lines[0]) != string(body) {
			t.Errorf("%s gave %d lines, want the body as one plain line", name, len(lines))
		}
	}
}

func TestBracketsInsideStringsDoNotCountAsNesting(t *testing.T) {
	body := []byte(`{"a":"` + strings.Repeat(`[\"[`, 400) + `","b":1}`)
	if got := Compact(body, 20); len(got) < 2 {
		t.Fatalf("a shallow body with bracket-heavy strings was shown plain: %d line", len(got))
	}
}

// FuzzCompact: any body, JSON or not, at any width.
func FuzzCompact(fuzz *testing.F) {
	for _, seed := range []string{
		`{"a":1,"b":[1,2,3],"c":{"d":null}}`, `[[[[]]]]`, `{"k":"日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語"}`,
		`{"a":`, `plain text`, ``, `"just a string"`, `{"a":1e999}`,
	} {
		fuzz.Add([]byte(seed), 20)
	}
	fuzz.Fuzz(func(t *testing.T, body []byte, width int) {
		_ = Compact(body, width%200)
		_ = Lines(body)
	})
}
