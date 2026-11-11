package widget

import (
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// TabItem is one tab; Badge > 0 shows a count after the label.
type TabItem struct {
	Label string
	Badge int
}

// Tabs is a strip of labels with a gold bar under the active one.
func Tabs(items []TabItem, active int, opts ...Option) *El {
	var strip *El
	strip = Row(with(append([]Item{PadL(2)}, tabItems(items, active, selector(&strip))...), opts)...)
	return strip
}

// selector is the click handler of the tabs of *strip: it runs the strip's
// OnSelect, which is set only after the tabs are built.
func selector(strip **El) func(i int) {
	return func(i int) {
		if (*strip).SelectHandler() != nil {
			(*strip).SelectHandler()(i)
		}
	}
}

// ScrollTabs is Tabs in a strip that scrolls sideways when the labels do
// not fit, keeping the active tab in view. It grows to the free width.
func ScrollTabs(state *ScrollState, items []TabItem, active int, opts ...Option) *El {
	var strip *El
	strip = HScroll(state, with(append([]Item{Grow(1), PadL(2)}, tabItems(items, active, selector(&strip))...), opts)...)
	return strip
}

func tabItems(items []TabItem, active int, onSelect func(i int)) []Item {
	out := make([]Item, len(items))
	for i, it := range items {
		i, it := i, it
		tab := []Item{
			Gap(4), PadX(10), AlignItems(AlignBaseline),
			OnClick(func() { onSelect(i) }),
			When(i > 0, Separator(10)),
			Text(it.Label, Color(tabColor(i == active))),
			If(it.Badge > 0, Text(strconv.Itoa(it.Badge), Small(), Muted())),
		}
		if i == active { // outlined top, left and right in the accent and open at the bottom, where the strip's line runs under the others
			tab = append(tab, ActiveTab())
		}
		out[i] = Row(tab...)
	}
	return out
}

// ActiveTab is the look of the open tab: the pane background, outlined in
// the accent on top, left and right and open at the bottom, where the
// strip's line runs under the other tabs; the strip scrolls it into view.
// Tabs apply it to their active item; a hand-built tab (one with handlers
// Tabs does not offer) uses it the same way.
func ActiveTab() Option {
	return func(tab *El) {
		for _, option := range []Option{Bg(core.Bg), EdgeT(core.AccentLine), EdgeL(core.AccentLine), EdgeR(core.AccentLine), KeepInView()} {
			option(tab)
		}
	}
}

func tabColor(active bool) core.Token {
	if active {
		return core.Text
	}
	return core.Muted
}
