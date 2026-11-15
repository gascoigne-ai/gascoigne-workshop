package filedialog

import (
	"path/filepath"
	"slices"
	"strings"
)

// zenityLike is a program that takes zenity's command line. matedialog is
// MATE's fork of zenity and qarma a Qt rewrite; both read the same
// --file-selection options and exit 1 when the user cancels.
func zenityLike(name string) tool {
	return tool{
		name: name,
		open: func(title string) []string { return []string{"--file-selection", "--title=" + title} },
		save: func(title, name string) []string {
			return []string{"--file-selection", "--save", "--confirm-overwrite", "--title=" + title, "--filename=" + name}
		},
		cancels: []int{1},
	}
}

var (
	zenity     = zenityLike("zenity")
	matedialog = zenityLike("matedialog")
	qarma      = zenityLike("qarma")
	kdialog    = tool{
		name:    "kdialog",
		open:    func(title string) []string { return []string{"--title", title, "--getopenfilename", "."} },
		save:    func(title, name string) []string { return []string{"--title", title, "--getsavefilename", name} },
		cancels: []int{1},
	}
	// yad's file dialog is --file; Escape or the window's close button
	// makes it exit 252, its Cancel button 1.
	yad = tool{
		name: "yad",
		open: func(title string) []string { return []string{"--file", "--title=" + title} },
		save: func(title, name string) []string {
			return []string{"--file", "--save", "--confirm-overwrite", "--title=" + title, "--filename=" + name}
		},
		cancels: []int{1, 252},
	}
)

// desktop is a family of desktops and the dialog programs it tries, best
// first. A desktop's own program comes first; a fork of zenity a desktop
// ships (matedialog, qarma) is tried before zenity when it is installed,
// but the user is told to install the program they will find packaged
// everywhere (suggest).
type desktop struct {
	names   []string // as the desktop variables hold them, lower case, without "x-"
	order   []tool
	suggest string
}

var desktops = []desktop{
	{[]string{"kde", "plasma", "plasmawayland", "plasmax11"}, []tool{kdialog, qarma, zenity, yad, matedialog}, "kdialog"},
	{[]string{"mate"}, []tool{matedialog, zenity, yad, kdialog, qarma}, "zenity"},
	{[]string{"lxqt"}, []tool{qarma, zenity, kdialog, yad, matedialog}, "zenity"},
	{[]string{"gnome", "gnome-classic", "gnome-flashback", "ubuntu", "unity", "cinnamon", "budgie", "budgie-desktop", "pantheon", "xfce", "lxde"},
		[]tool{zenity, yad, kdialog, qarma, matedialog}, "zenity"},
}

// anyDesktop is the order of a desktop with no row: a tiling window
// manager, WSL, or no desktop variable at all.
var anyDesktop = desktop{nil, []tool{zenity, kdialog, yad, qarma, matedialog}, "zenity or kdialog"}

// desktopNames are the names the environment gives the desktop, most
// specific first: XDG_CURRENT_DESKTOP is a colon-separated list
// ("ubuntu:GNOME"), DESKTOP_SESSION is sometimes the path of a session file.
func desktopNames(getenv func(string) string) []string {
	names := strings.Split(getenv("XDG_CURRENT_DESKTOP"), ":")
	names = append(names, getenv("XDG_SESSION_DESKTOP"))
	if session := getenv("DESKTOP_SESSION"); session != "" {
		names = append(names, filepath.Base(session))
	}
	return names
}

// desktopOf is the row of the first desktop name that has one.
func desktopOf(getenv func(string) string) desktop {
	for _, name := range desktopNames(getenv) {
		name = strings.TrimPrefix(strings.ToLower(name), "x-")
		for _, row := range desktops {
			if slices.Contains(row.names, name) {
				return row
			}
		}
	}
	return anyDesktop
}

// preferred is the order in which system goos tries the dialog programs,
// whether or not they are installed.
func preferred(goos string, getenv func(string) string) []tool {
	if list, ok := toolsBySystem[goos]; ok {
		return list
	}
	return desktopOf(getenv).order
}

// suggested is the program to install for native dialogs, "" on a system
// whose own dialog is always there.
func suggested(goos string, getenv func(string) string) string {
	if _, ok := toolsBySystem[goos]; ok {
		return ""
	}
	return desktopOf(getenv).suggest
}

// installed is the programs of list that lookPath finds, in order.
func installed(list []tool, lookPath func(string) (string, error)) []found {
	var result []found
	for _, each := range list {
		if path, err := lookPath(each.name); err == nil {
			result = append(result, found{each, path})
		}
	}
	return result
}
