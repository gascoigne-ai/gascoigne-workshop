package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func TestStatusPillsFollowTheDashboardBadges(t *testing.T) {
	for status, want := range map[int][2]widget.Token{
		200: {widget.OK, widget.OKSoft}, 204: {widget.OK, widget.OKSoft},
		301: {widget.Warn, widget.WarnSoft}, 404: {widget.Error, widget.ErrorSoft}, 500: {widget.Error, widget.ErrorSoft},
	} {
		if fg, bg := statusPill(status); fg != want[0] || bg != want[1] {
			t.Errorf("%d: %v on %v, want %v on %v", status, fg, bg, want[0], want[1])
		}
	}
	for _, id := range []string{"dark", "light"} {
		th := appearance.ThemeFor(id, "medium")
		for _, status := range []int{200, 301, 404} {
			fg, bg := statusPill(status)
			if r := widget.ContrastRatio(th.Color(fg), th.Color(bg)); r < 3.5 {
				t.Errorf("%s: pill %d text %.2f:1 on its background", id, status, r)
			}
		}
	}
}
