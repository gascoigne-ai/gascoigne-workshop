package core

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ellipsizeByDropping is the first Ellipsize: it drops one rune per step.
// It stays here as the oracle the faster search must agree with.
func ellipsizeByDropping(c *Ctx, face *text.GoTextFace, content string, maxW int) string {
	if c.Width(face, content) <= maxW {
		return content
	}
	runes := []rune(content)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if cand := strings.TrimRight(string(runes), " ") + "..."; c.Width(face, cand) <= maxW {
			return cand
		}
	}
	return ""
}

func TestEllipsizeShortensToFit(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	face := ctx.F.UI
	full := "a long request name"
	tests := []struct {
		name    string
		content string
		maxW    int
		want    string
	}{
		{"empty fits", "", 10, ""},
		{"fits", full, ctx.Width(face, full), full},
		{"nothing fits", full, 1, ""},
		{"only the dots fit", full, ctx.Width(face, "..."), "..."},
		{"trailing space is trimmed before the dots", "ab cd", ctx.Width(face, "ab...") + 1, "ab..."},
		{"multibyte", "żółć gęślą jaźń", ctx.Width(face, "żółć g...") + 1, "żółć g..."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ctx.Ellipsize(face, test.content, test.maxW); got != test.want {
				t.Errorf("Ellipsize(%q, %d) = %q, want %q", test.content, test.maxW, got, test.want)
			}
		})
	}
}

func TestEllipsizeAgreesWithDroppingOneRuneAtATime(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	face := ctx.F.UI
	alphabet := []rune("abcij  m  w    .,ż字 WI")
	rng := rand.New(rand.NewSource(133))
	for n := 0; n < 10000; n++ {
		runes := make([]rune, rng.Intn(40))
		for i := range runes {
			runes[i] = alphabet[rng.Intn(len(alphabet))]
		}
		content := string(runes)
		maxW := rng.Intn(ctx.Width(face, content) + 20)
		want := ellipsizeByDropping(ctx, face, content, maxW)
		if got := ctx.Ellipsize(face, content, maxW); got != want {
			t.Fatalf("Ellipsize(%q, %d) = %q, want %q", content, maxW, got, want)
		}
	}
}
