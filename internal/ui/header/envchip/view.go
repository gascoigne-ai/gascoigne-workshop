package envchip

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var template = markup.Must("envchip.vml", source)

// View is the chip; clicking it opens the menu where the pointer is.
func (c *Component) View(ctx *w.Ctx) Node {
	return template.MustBuild(sizes.Scope(ctx, markup.Scope{
		"label":    c.label(),
		"openMenu": func(posX, posY int) { c.deps.Menus.Open(posX, posY, c.menu()) },
	}))
}

// Template, parsed once above.
const (
	source = `<!-- The chip stops at 200 px and cuts a long name, so the settings button
     beside it stays in a 400 px window; "No environment" (111 px) fits.
     Narrower, it shrinks after the title, down to its padding and a few
     pixels: a larger min-w would widen the chip of a short name. -->
<row pad-x="{{labelPad}}" h="{{compactH}}" max-w="{{envMaxW}}" min-w="{{envMinW}}" shrink bg="hero-text" control="pill" align-items="center" on-click-at="openMenu" tooltip="Environment: variables that override the collection's">
  <text small bold ellipsis grow="1" color="hero">{{label}}</text>
</row>
`
)
