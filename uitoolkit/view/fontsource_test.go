package view

import (
	"strings"
	"testing"

	"bytes"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"
)

func monoSource(t *testing.T) *text.GoTextFaceSource {
	t.Helper()
	source, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func TestTextWithAFontSourceIsMeasuredInThatFont(t *testing.T) {
	c := testCtx(t)
	plainText := Text("iiiiii")
	mono := Text("iiiiii", FontSource(monoSource(t)))
	Layout(c, Row(plainText, mono), 600, 30)
	if mono.rect.W <= plainText.rect.W {
		t.Fatalf("a monospace \"i\" is wider than a proportional one: %d vs %d", mono.rect.W, plainText.rect.W)
	}
}

func TestADumpMarksTextDrawnInAnotherFont(t *testing.T) {
	dump := Dump(Row(Text("a"), Text("b", FontSource(monoSource(t)))))
	if strings.Count(dump, "font=") != 1 {
		t.Fatalf("dump:\n%s", dump)
	}
}
