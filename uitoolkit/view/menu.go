package view

import (
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

const (
	// MenuMinW and MenuMaxW bound the width of a menu, which is otherwise
	// as wide as its widest row.
	MenuMinW = 120
	MenuMaxW = 420

	// MenuRowH, MenuSepH and MenuPad are the height of a plain row, of a
	// separator and the inset around the rows, for callers that position
	// things by the rows of a menu.
	MenuRowH = 24
	MenuSepH = 9
	MenuPad  = 4

	menuHintH   = 16 // the extra height of each hint line
	menuRowPadX = 10
	menuIconW   = 14 // the icon column; rows without an icon keep it so labels line up
)

// MenuItem is one row of a Menu. A separator has no label. Icon is the
// name of a registered icon, "" for none.
type MenuItem struct {
	Label, Shortcut string
	Hint            string // muted lines under the label saying what the item is
	Icon            string
	Disabled        bool
	Separator       bool
	Run             func()
}

// Selectable reports whether the row can be highlighted and picked.
func (item MenuItem) Selectable() bool { return !item.Separator && !item.Disabled }

// MenuState is an open menu: where it opens, its rows and the row the
// keyboard has highlighted (-1 when no row can be picked).
type MenuState struct {
	X, Y  int
	Items []MenuItem
	Hot   int
}

// NewMenu is a menu at x, y with its first selectable row highlighted.
func NewMenu(x, y int, items []MenuItem) *MenuState {
	return &MenuState{X: x, Y: y, Items: items, Hot: NextSelectable(items, -1, 1)}
}

// NextSelectable is the index of the next enabled, non-separator row
// after from in direction dir (1 or -1), wrapping at the ends; -1 when
// no row can be picked.
func NextSelectable(items []MenuItem, from, dir int) int {
	count := len(items)
	for step := 1; step <= count; step++ {
		index := ((from+dir*step)%count + count) % count
		if items[index].Selectable() {
			return index
		}
	}
	return -1
}

// Keys moves the highlight with Up and Down and returns the highlighted
// row when Enter went down this frame, else -1.
func (menu *MenuState) Keys(input core.Input) int {
	if input.Pressed(ebiten.KeyArrowDown) {
		menu.Hot = NextSelectable(menu.Items, menu.Hot, 1)
	}
	if input.Pressed(ebiten.KeyArrowUp) {
		menu.Hot = NextSelectable(menu.Items, menu.Hot, -1)
	}
	if (input.Enter || input.Pressed(ebiten.KeyEnter)) && menu.Hot >= 0 {
		return menu.Hot
	}
	return -1
}

// Menu is a popup menu at its state's position: a modal overlay that is as
// wide as its widest row. OnSelect runs with the index of a clicked row
// that can be picked, OnDismiss when the user clicks outside it. The
// context only measures text.
func Menu(c *core.Ctx, state *MenuState, opts ...Option) *El {
	var menu *El
	pick := func(index int) {
		if menu.onSelect != nil && state.Items[index].Selectable() {
			menu.onSelect(index)
		}
	}
	rows := make([]Item, len(state.Items))
	for index, item := range state.Items {
		rows[index] = menuRow(state, index, item, pick)
	}
	menu = Col(with(append([]Item{
		W(MenuWidth(c, state.Items)), Pad(MenuPad), Bg(core.Bg), Control("menu"), Shadow(),
		FloatAt(state.X, state.Y), Modal(),
	}, rows...), opts)...)
	return menu
}

func menuRow(state *MenuState, index int, item MenuItem, pick func(int)) *El {
	if item.Separator {
		return Row(H(MenuSepH), PadX(6), HRule())
	}
	color := core.Text
	if item.Disabled {
		color = core.Muted
	}
	look := HoverBg(core.Hover)
	switch {
	case item.Disabled:
		look = Gap(0)
	case index == state.Hot:
		look = Bg(core.Selected)
	}
	var icon Item
	if menuHasIcons(state.Items) {
		icon = Spacer(W(menuIconW))
		if item.Icon != "" {
			icon = Icon(item.Icon, W(menuIconW), Color(color))
		}
	}
	label := Item(Text(item.Label, Color(color), Grow(1)))
	height := MenuRowH
	if item.Hint != "" {
		lines := strings.Split(item.Hint, "\n")
		parts := []Item{Grow(1), Text(item.Label, Color(color))}
		for _, line := range lines {
			parts = append(parts, Text(line, Small(), Muted(), Ellipsis()))
		}
		label = Col(parts...)
		height += menuHintH * len(lines)
	}
	return Row(H(height), PadX(menuRowPadX), Gap(8), AlignItems(AlignCenter), Control("row"), look, OnClick(func() { pick(index) }),
		icon,
		label,
		If(item.Shortcut != "", Text(core.PlatformKeyText(runtime.GOOS, item.Shortcut), Small(), Muted())),
	)
}

func menuHasIcons(items []MenuItem) bool {
	for _, item := range items {
		if !item.Separator && item.Icon != "" {
			return true
		}
	}
	return false
}

// MenuWidth is the width of a menu of items: its widest row within
// MenuMinW and MenuMaxW.
func MenuWidth(c *core.Ctx, items []MenuItem) int {
	widest := 0
	icons := menuHasIcons(items)
	for _, item := range items {
		if item.Separator {
			continue
		}
		row := c.Width(c.F.UI, item.Label)
		for _, line := range strings.Split(item.Hint, "\n") {
			row = max(row, c.Width(c.F.Small, line))
		}
		if item.Shortcut != "" {
			row += 16 + c.Width(c.F.Small, core.PlatformKeyText(runtime.GOOS, item.Shortcut))
		}
		if icons {
			row += menuIconW + 8
		}
		widest = max(widest, row)
	}
	return min(max(widest+2*menuRowPadX+2*MenuPad, MenuMinW), MenuMaxW)
}

// OpenMenu opens a popup menu at x, y; it replaces one that is open. Mount
// PopupMenuLayer in the window's Stack to show it and call PopupMenuKeys
// each frame for its keyboard.
func OpenMenu(c *core.Ctx, x, y int, items []MenuItem) { c.Menu = NewMenu(x, y, items) }

// CloseMenu closes the popup menu, if one is open.
func CloseMenu(c *core.Ctx) { c.Menu = nil }

// MenuOpen reports whether a popup menu is open.
func MenuOpen(c *core.Ctx) bool { return PopupMenu(c) != nil }

// PopupMenu is the open popup menu, nil when there is none.
func PopupMenu(c *core.Ctx) *MenuState {
	state, _ := c.Menu.(*MenuState)
	return state
}

// PopupMenuLayer is the open popup menu as an overlay, nil when none is
// open. A pick or a click outside closes it; a pick then runs the row's
// action, so the action may open another menu.
func PopupMenuLayer(c *core.Ctx) Node {
	state := PopupMenu(c)
	if state == nil {
		return nil
	}
	return Menu(c, state, OnSelect(func(index int) { pickMenuRow(c, index) }), OnDismiss(func() { CloseMenu(c) }))
}

// PopupMenuKeys drives the open popup menu with the frame's keys: Up and
// Down move the highlight, Enter picks it. Without an open menu it does
// nothing.
func PopupMenuKeys(c *core.Ctx, input core.Input) {
	state := PopupMenu(c)
	if state == nil {
		return
	}
	if picked := state.Keys(input); picked >= 0 {
		pickMenuRow(c, picked)
	}
}

func pickMenuRow(c *core.Ctx, index int) {
	state := PopupMenu(c)
	if state == nil || !state.Items[index].Selectable() {
		return
	}
	run := state.Items[index].Run
	CloseMenu(c)
	if run != nil {
		run()
	}
}
