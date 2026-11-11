package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestIconHoverColorOverridesTheIconsOwnHoverToken(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		hovered bool
		want    core.Token
	}{
		{"resting uses Color", []Option{Color(core.HeroMuted), HoverColor(core.HeroText)}, false, core.HeroMuted},
		{"hovered uses HoverColor", []Option{Color(core.HeroMuted), HoverColor(core.HeroText)}, true, core.HeroText},
		{"hovered without HoverColor keeps the icon's own token", []Option{Color(core.HeroMuted)}, true, core.Text},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			icon, _ := lookupIcon("dots")
			if got := iconToken(Icon("dots", test.opts...), icon, test.hovered); got != test.want {
				t.Fatalf("token = %q, want %q", got, test.want)
			}
		})
	}
}
