package filedialog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// envOf is a getenv over a fixed environment.
func envOf(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

// everything finds every program, as on a machine with all of them installed.
func everything(name string) (string, error) { return "/usr/bin/" + name, nil }

// only finds the named programs.
func only(names ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		if slices.Contains(names, name) {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

// noDesktop clears the desktop variables, so a test reads the same on any
// desktop it runs on.
func noDesktop(t *testing.T) {
	t.Helper()
	for _, key := range []string{"XDG_CURRENT_DESKTOP", "XDG_SESSION_DESKTOP", "DESKTOP_SESSION"} {
		t.Setenv(key, "")
	}
}

func namesOf(list []found) []string {
	var names []string
	for _, each := range list {
		names = append(names, each.tool.name)
	}
	return names
}

func TestTheDesktopOrdersTheDialogPrograms(t *testing.T) {
	gtkFirst := []string{"zenity", "yad", "kdialog", "qarma", "matedialog"}
	kdeFirst := []string{"kdialog", "qarma", "zenity", "yad", "matedialog"}
	anyFirst := []string{"zenity", "kdialog", "yad", "qarma", "matedialog"}
	cases := []struct {
		name    string
		env     map[string]string
		want    []string
		suggest string
	}{
		{"GNOME", map[string]string{"XDG_CURRENT_DESKTOP": "GNOME"}, gtkFirst, "zenity"},
		{"Ubuntu's GNOME names itself first", map[string]string{"XDG_CURRENT_DESKTOP": "ubuntu:GNOME"}, gtkFirst, "zenity"},
		{"Budgie", map[string]string{"XDG_CURRENT_DESKTOP": "Budgie:GNOME"}, gtkFirst, "zenity"},
		{"Cinnamon", map[string]string{"XDG_CURRENT_DESKTOP": "X-Cinnamon"}, gtkFirst, "zenity"},
		{"Pantheon", map[string]string{"XDG_CURRENT_DESKTOP": "Pantheon"}, gtkFirst, "zenity"},
		{"XFCE", map[string]string{"XDG_CURRENT_DESKTOP": "XFCE"}, gtkFirst, "zenity"},
		{"KDE", map[string]string{"XDG_CURRENT_DESKTOP": "KDE"}, kdeFirst, "kdialog"},
		{"Plasma by its session only", map[string]string{"DESKTOP_SESSION": "plasmawayland"}, kdeFirst, "kdialog"},
		{"Plasma by a session path", map[string]string{"DESKTOP_SESSION": "/usr/share/xsessions/plasma"}, kdeFirst, "kdialog"},
		{"MATE tries its own zenity first", map[string]string{"XDG_CURRENT_DESKTOP": "MATE"}, []string{"matedialog", "zenity", "yad", "kdialog", "qarma"}, "zenity"},
		{"LXQt tries the Qt zenity first", map[string]string{"XDG_CURRENT_DESKTOP": "LXQt"}, []string{"qarma", "zenity", "kdialog", "yad", "matedialog"}, "zenity"},
		{"the session desktop when the current one has no row", map[string]string{"XDG_CURRENT_DESKTOP": "sway", "XDG_SESSION_DESKTOP": "KDE"}, kdeFirst, "kdialog"},
		{"Sway", map[string]string{"XDG_CURRENT_DESKTOP": "sway"}, anyFirst, "zenity or kdialog"},
		{"Hyprland", map[string]string{"XDG_CURRENT_DESKTOP": "Hyprland"}, anyFirst, "zenity or kdialog"},
		{"i3 with no desktop variable", map[string]string{"DISPLAY": ":0"}, anyFirst, "zenity or kdialog"},
		{"WSL", map[string]string{"WSL_DISTRO_NAME": "Ubuntu", "WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, anyFirst, "zenity or kdialog"},
		{"empty environment", map[string]string{}, anyFirst, "zenity or kdialog"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := namesOf(installed(preferred("linux", envOf(tc.env)), everything))
			if !slices.Equal(got, tc.want) {
				t.Fatalf("order %v, want %v", got, tc.want)
			}
			if got := suggested("linux", envOf(tc.env)); got != tc.suggest {
				t.Fatalf("suggested %q, want %q", got, tc.suggest)
			}
		})
	}
}

func TestOnlyInstalledProgramsAreCandidates(t *testing.T) {
	mate := envOf(map[string]string{"XDG_CURRENT_DESKTOP": "MATE"})
	got := namesOf(installed(preferred("linux", mate), only("zenity", "kdialog")))
	if !slices.Equal(got, []string{"zenity", "kdialog"}) {
		t.Fatalf("MATE without matedialog: %v", got)
	}
	if got := installed(preferred("linux", mate), only()); len(got) != 0 {
		t.Fatalf("nothing installed: %v", namesOf(got))
	}
}

func TestMacAndWindowsIgnoreTheDesktopAndSuggestNothing(t *testing.T) {
	kde := envOf(map[string]string{"XDG_CURRENT_DESKTOP": "KDE"})
	if got := namesOf(installed(preferred("darwin", kde), everything)); got[0] != "osascript" {
		t.Fatalf("darwin order %v", got)
	}
	if got := namesOf(installed(preferred("windows", kde), everything)); !slices.Equal(got, []string{"powershell"}) {
		t.Fatalf("windows order %v", got)
	}
	for _, system := range []string{"darwin", "windows"} {
		if got := suggested(system, kde); got != "" {
			t.Fatalf("%s suggests %q, but its dialog is always there", system, got)
		}
	}
}

func TestEachLinuxProgramGetsItsOwnCommandLine(t *testing.T) {
	type lines struct{ open, save []string }
	zenityHome := lines{
		[]string{"--file-selection", "--title=Pick", "--filename=/home/me/"},
		[]string{"--file-selection", "--save", "--confirm-overwrite", "--title=Save", "--filename=/home/me/out.json"},
	}
	// without a home folder the dialogs start where they did before: the
	// program's own choice for open, the bare name for save
	zenityNoHome := lines{
		[]string{"--file-selection", "--title=Pick"},
		[]string{"--file-selection", "--save", "--confirm-overwrite", "--title=Save", "--filename=out.json"},
	}
	cases := []struct {
		name         string
		home, noHome lines
		cancels      []int
	}{
		// zenity and matedialog read a --filename ending in / as the folder to start in
		{"zenity", zenityHome, zenityNoHome, []int{1}},
		{"matedialog", zenityHome, zenityNoHome, []int{1}},
		// qarma takes a --filename ending in /. as the folder, anything else as a file to select
		{"qarma", lines{
			[]string{"--file-selection", "--title=Pick", "--filename=/home/me/."},
			zenityHome.save,
		}, zenityNoHome, []int{1}},
		// kdialog's start argument is a folder, or a file whose folder it opens with the name filled in
		{"kdialog", lines{
			[]string{"--title", "Pick", "--getopenfilename", "/home/me"},
			[]string{"--title", "Save", "--getsavefilename", "/home/me/out.json"},
		}, lines{
			[]string{"--title", "Pick", "--getopenfilename", "."},
			[]string{"--title", "Save", "--getsavefilename", "out.json"},
		}, []int{1}},
		// yad's file dialog is --file (its man page's --file-selection example is not an option);
		// it exits 252 when closed with Escape or the window's button
		{"yad", lines{
			[]string{"--file", "--title=Pick", "--filename=/home/me/"},
			[]string{"--file", "--save", "--confirm-overwrite", "--title=Save", "--filename=/home/me/out.json"},
		}, lines{
			[]string{"--file", "--title=Pick"},
			[]string{"--file", "--save", "--confirm-overwrite", "--title=Save", "--filename=out.json"},
		}, []int{1, 252}},
	}
	all := installed(preferred("linux", envOf(nil)), everything)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			index := slices.IndexFunc(all, func(each found) bool { return each.tool.name == tc.name })
			if index < 0 {
				t.Fatalf("%s is not in the table", tc.name)
			}
			program := all[index].tool
			for _, start := range []struct {
				home string
				want lines
			}{{"/home/me", tc.home}, {"", tc.noHome}} {
				if got := program.open("Pick", start.home); !slices.Equal(got, start.want.open) {
					t.Errorf("open from %q: %q, want %q", start.home, got, start.want.open)
				}
				if got := program.save("Save", start.home, "out.json"); !slices.Equal(got, start.want.save) {
					t.Errorf("save from %q: %q, want %q", start.home, got, start.want.save)
				}
			}
			if !slices.Equal(program.cancels, tc.cancels) {
				t.Errorf("cancel codes %v, want %v", program.cancels, tc.cancels)
			}
		})
	}
}

func TestASaveNameThatIsAPathIsKept(t *testing.T) {
	if got := savePath("/home/me", "/tmp/out.json"); got != "/tmp/out.json" {
		t.Fatalf("absolute name: %q", got)
	}
	if got := savePath("/home/me", ""); got != "/home/me/" {
		t.Fatalf("no name starts in the folder: %q", got)
	}
}

func TestClosingYadWithEscapeIsACancel(t *testing.T) {
	goos = "linux"
	t.Cleanup(func() { goos = runtime.GOOS })
	noDesktop(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "yad"), []byte("#!/bin/sh\nexit 252\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := Open(context.Background(), "x"); err != ErrCancelled {
		t.Fatalf("err = %v", err)
	}
}
