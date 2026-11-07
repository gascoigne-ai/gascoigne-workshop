package ui

import "strings"

// menuIcons maps the first word of a menu label (or its first two words,
// which win) to the icon shown before it.
var menuIcons = map[string]string{
	"Copy": "copy", "Duplicate": "copy",
	"Cut": "scissors", "Paste": "clipboard", "Import": "clipboard",
	"Delete": "trash", "Rename": "pencil", "Close": "cross", "Find": "search",
	"New": "plus", "Insert": "plus",
	"Move up": "chevron-up", "Move down": "chevron-down",
	"Save": "save", "Export": "save", "Open": "external",
	"Undo": "undo", "Redo": "redo",
}

// menuIconW is the width of the icon column of a menu row; rows without an
// icon keep the space so the labels line up.
const menuIconW = 14

// menuIcon is the icon of an item: its own, else the one its label starts
// with, else none.
func menuIcon(item menuItem) string {
	if item.icon != "" {
		return item.icon
	}
	words := strings.Fields(strings.TrimRight(item.label, "."))
	if len(words) >= 2 {
		if name, ok := menuIcons[words[0]+" "+words[1]]; ok {
			return name
		}
	}
	if len(words) >= 1 {
		return menuIcons[words[0]]
	}
	return ""
}
