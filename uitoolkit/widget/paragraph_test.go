package widget

import (
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const paragraphText = "Words that are long enough to need several lines when the room is narrow, and one when it is wide."

// A paragraph breaks at the width its parent gives it: the narrower, the
// taller, and never wider than the room.
func TestAParagraphWrapsToTheWidthItIsGiven(t *testing.T) {
	c := testCtx(t)
	heights := map[int]int{}
	for _, width := range []int{120, 240, 2000} {
		paragraph := Paragraph(paragraphText)
		Layout(c, Col(AlignItems(AlignStart), paragraph), width, 400)
		rect := paragraph.Rect()
		if rect.W > width {
			t.Fatalf("width %d: paragraph is %d wide", width, rect.W)
		}
		heights[width] = rect.H
	}
	if !(heights[120] > heights[240] && heights[240] > heights[2000]) {
		t.Fatalf("heights by width %v: a narrower room must give a taller paragraph", heights)
	}
}

func TestAParagraphHoldsAllItsWords(t *testing.T) {
	c := testCtx(t)
	paragraph := Paragraph(paragraphText)
	Layout(c, Col(paragraph), 150, 400)
	got := strings.Join(wrapParagraph(c, c.F.Small, paragraphText, 150), " ")
	if got != paragraphText {
		t.Fatalf("lines joined %q, want %q", got, paragraphText)
	}
}

func TestAParagraphWithNoWordsHasNoHeight(t *testing.T) {
	c := testCtx(t)
	paragraph := Paragraph("")
	Layout(c, Col(AlignItems(AlignStart), paragraph), 150, 400)
	if paragraph.Rect().H != 0 {
		t.Fatalf("empty paragraph is %d high", paragraph.Rect().H)
	}
}

func TestADumpShowsTheTextOfAParagraph(t *testing.T) {
	if got := props(Paragraph(paragraphText)); !strings.Contains(got, `text="`+paragraphText+`"`) {
		t.Fatalf("dump %q does not show the text", got)
	}
}
