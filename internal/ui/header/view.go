package header

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var template = markup.Must("header.vml", source)

// View is the banner: the collection's name, the environment chip and the
// settings button.
func (c *Component) View(ctx *w.Ctx) Node {
	return sizes.Build(ctx, template, markup.Scope{
		"headerH":     sizes.HeaderH,
		"gutter":      w.Gutter,
		"title":       c.deps.Title.Title(),
		"env":         c.chip.View(ctx),
		"toggleMenu":  c.deps.Settings.Toggle,
		"settingsTip": c.deps.Hints.Hint("Settings", "settings"),
	})
}

// Template, parsed once above.
const (
	source = `<!-- In a window too narrow for everything the title gives up room first,
     down to 48 px, then the chip, so the settings button stays inside. -->
<row h="{{headerH}}" pad-x="{{gutter}}" bg="hero" control="hero">
  <text bold ellipsis shrink min-w="{{titleMinW}}" color="hero-text">{{title}}</text>
  <spacer/>
  <slot name="env"/>
  <spacer w="{{gutter}}"/>
  <row w="{{headerH}}" on-click="toggleMenu" tooltip="{{settingsTip}}">
    <icon name="gear" grow="1" color="hero-muted" hover-color="hero-text"/>
  </row>
</row>
`
)
