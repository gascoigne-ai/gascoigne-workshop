package filebrowser

import (
	"errors"
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
)

// home is a folder listing like a real one: folders and files, hidden
// ones, and names that differ only in case.
var home = []workspace.FolderEntry{
	{Name: "notes.txt"},
	{Name: "Projects", Dir: true},
	{Name: ".config", Dir: true},
	{Name: "api.json"},
	{Name: "bin", Dir: true},
	{Name: ".profile"},
	{Name: "Body.json"},
}

// shown opens a browser in mode on /home/me listed as home.
func shown(t *testing.T, mode Mode, name string) Model {
	t.Helper()
	model, effect := New(mode, "/home/me", "/home/me", name)
	if effect.List != "/home/me" {
		t.Fatalf("a new browser lists its start folder first: %+v", effect)
	}
	model, _ = Update(model, Listed{Dir: "/home/me", Entries: home})
	return model
}

func names(entries []workspace.FolderEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name
	}
	return out
}

func TestFoldersComeFirstThenNamesIgnoringCaseAndHiddenOnesAreLeftOut(t *testing.T) {
	model := shown(t, Open, "")
	want := []string{"bin", "Projects", "api.json", "Body.json", "notes.txt"}
	if got := names(model.Visible()); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	model, _ = Update(model, HiddenToggled{})
	want = []string{".config", "bin", "Projects", ".profile", "api.json", "Body.json", "notes.txt"}
	if got := names(model.Visible()); !slices.Equal(got, want) {
		t.Fatalf("with hidden files: got %q, want %q", got, want)
	}
}

func TestTypingFiltersTheOpenList(t *testing.T) {
	cases := []struct {
		name  string
		typed string
		want  []string
	}{
		{"nothing typed shows all", "/home/me/", []string{"bin", "Projects", "api.json", "Body.json", "notes.txt"}},
		{"a part of the name, any case", "/home/me/JSON", []string{"api.json", "Body.json"}},
		{"an extension pattern", "/home/me/*.txt", []string{"notes.txt"}},
		{"a pattern ignores case", "/home/me/b*", []string{"bin", "Body.json"}},
		{"a leading dot shows hidden names", "/home/me/.c", []string{".config"}},
		{"a relative name filters too", "pro", []string{"Projects"}},
		{"a path into another folder does not filter this one", "/etc/ho", []string{"bin", "Projects", "api.json", "Body.json", "notes.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model, _ := Update(shown(t, Open, ""), Typed{Text: tc.typed})
			if got := names(model.Visible()); !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSaveDoesNotFilterByTheName(t *testing.T) {
	model := shown(t, Save, "body.json")
	if model.Text != "/home/me/body.json" {
		t.Fatalf("the save name follows the folder: %q", model.Text)
	}
	if len(model.Visible()) != 5 {
		t.Fatalf("the name typed for saving is not a filter: %q", names(model.Visible()))
	}
}

func TestTypingSelectsTheExactNameElseTheFirstMatch(t *testing.T) {
	cases := []struct {
		typed string
		want  int // index in Visible, -1 for none
	}{
		{"/home/me/", -1},
		{"/home/me/o", 0},     // Projects, Body.json, notes.txt: the first
		{"/home/me/.json", 0}, // api.json and Body.json hold it
		{"/home/me/zzz", -1},  // no match
		{"/etc/hosts", -1},    // another folder: Enter goes there
	}
	for _, tc := range cases {
		model, _ := Update(shown(t, Open, ""), Typed{Text: tc.typed})
		if model.Selected != tc.want {
			t.Errorf("typed %q: selected %d, want %d (%q)", tc.typed, model.Selected, tc.want, names(model.Visible()))
		}
	}
	// The exact name wins over a longer one listed before it, so Enter
	// opens what was typed.
	model, _ := New(Open, "/d", "", "")
	model, _ = Update(model, Listed{Dir: "/d", Entries: []workspace.FolderEntry{{Name: "a.json"}, {Name: "a.json.d", Dir: true}}})
	model, _ = Update(model, Typed{Text: "/d/a.json"})
	if visible := model.Visible(); model.Selected < 0 || visible[model.Selected].Name != "a.json" {
		t.Fatalf("selected %d of %q", model.Selected, names(visible))
	}
	model, _ = Update(shown(t, Save, "x"), Typed{Text: "/home/me/api.json"})
	if model.Selected != -1 {
		t.Fatal("typing a save name selects nothing: Enter saves under the typed name")
	}
}

func TestUpAndDownMoveTheSelectionWithinTheList(t *testing.T) {
	model := shown(t, Open, "")
	steps := []struct {
		by   int
		want int
	}{{1, 0}, {1, 1}, {-1, 0}, {-1, 0}, {10, 4}, {1, 4}}
	for _, step := range steps {
		model, _ = Update(model, Moved{By: step.by})
		if model.Selected != step.want {
			t.Fatalf("by %d: selected %d, want %d", step.by, model.Selected, step.want)
		}
	}
	empty, _ := Update(shown(t, Open, ""), Typed{Text: "/home/me/zzz"})
	if empty, _ = Update(empty, Moved{By: 1}); empty.Selected != -1 {
		t.Fatalf("an empty list has nothing to select: %d", empty.Selected)
	}
}

func TestPickingAFileWhileSavingTakesItsName(t *testing.T) {
	model := shown(t, Save, "out.json")
	model, _ = Update(model, Picked{Index: 3}) // Body.json
	if model.Text != "/home/me/Body.json" {
		t.Fatalf("text %q", model.Text)
	}
	model, _ = Update(model, Picked{Index: 0}) // bin: a folder is not a name to save under
	if model.Text != "/home/me/Body.json" || model.Selected != 0 {
		t.Fatalf("text %q selected %d", model.Text, model.Selected)
	}
	open, _ := Update(shown(t, Open, ""), Picked{Index: 3})
	if open.Text != "/home/me/" {
		t.Fatalf("picking while opening leaves the filter alone: %q", open.Text)
	}
}

func TestEnterOpensChoosesOrAsks(t *testing.T) {
	cases := []struct {
		name    string
		mode    Mode
		typed   string
		move    int // rows to move down first, 0 for none
		want    Effect
		problem string
	}{
		{"a selected file is chosen", Open, "/home/me/api", 0, Effect{Chosen: "/home/me/api.json"}, ""},
		{"a selected folder is opened", Open, "/home/me/", 1, Effect{List: "/home/me/bin"}, ""},
		{"a typed folder is opened", Open, "/home/me/Projects", 0, Effect{List: "/home/me/Projects"}, ""},
		{"a typed path elsewhere lists its folder first", Open, "/etc/hosts", 0, Effect{List: "/etc"}, ""},
		{"a relative path is under the folder", Open, "Projects/a.go", 0, Effect{List: "/home/me/Projects"}, ""},
		{"a tilde is the home folder", Open, "~/bin/", 0, Effect{List: "/home/me/bin"}, ""},
		{"dot dot is the parent", Open, "/home/me/..", 0, Effect{List: "/home"}, ""},
		{"a missing file cannot be opened", Open, "/home/me/nope.txt", 0, Effect{}, "There is no nope.txt here"},
		{"nothing typed and nothing selected does nothing", Open, "/home/me/", 0, Effect{}, ""},
		{"a new name is saved", Save, "/home/me/new.json", 0, Effect{Chosen: "/home/me/new.json"}, ""},
		{"an existing name asks first", Save, "/home/me/api.json", 0, Effect{Chosen: "/home/me/api.json", Replace: true}, ""},
		{"a folder's name opens it and keeps no name", Save, "/home/me/bin", 0, Effect{List: "/home/me/bin"}, ""},
		{"an empty name is refused", Save, "/home/me/", 0, Effect{}, "Type a name for the file"},
		{"a name too long is refused", Save, "/home/me/" + long, 0, Effect{}, "The name is longer than 255 bytes"},
		{"a selected folder opens while saving", Save, "/home/me/x.json", 1, Effect{List: "/home/me/bin"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model, _ := Update(shown(t, tc.mode, ""), Typed{Text: tc.typed})
			if tc.move > 0 {
				model.Selected = -1
				model, _ = Update(model, Moved{By: tc.move})
			}
			model, effect := Update(model, Entered{})
			if effect != tc.want {
				t.Fatalf("effect %+v, want %+v", effect, tc.want)
			}
			if model.Problem != tc.problem {
				t.Fatalf("problem %q, want %q", model.Problem, tc.problem)
			}
		})
	}
}

var long = string(slices.Repeat([]byte("a"), 256))

func TestAPathElsewhereIsChosenOnceItsFolderIsListed(t *testing.T) {
	model, _ := Update(shown(t, Open, ""), Typed{Text: "/etc/hosts"})
	model, effect := Update(model, Entered{})
	if effect.List != "/etc" {
		t.Fatalf("effect %+v", effect)
	}
	model, effect = Update(model, Listed{Dir: "/etc", Entries: []workspace.FolderEntry{{Name: "hosts"}, {Name: "ssl", Dir: true}}})
	if effect.Chosen != "/etc/hosts" {
		t.Fatalf("the Enter goes on in the listed folder: %+v", effect)
	}
	if model.Dir != "/etc" {
		t.Fatalf("dir %q", model.Dir)
	}
}

func TestSavingKeepsTheNameWhileMovingBetweenFolders(t *testing.T) {
	model := shown(t, Save, "out.json")
	model, _ = Update(model, Moved{By: 1}) // bin
	model, effect := Update(model, Entered{})
	if effect.List != "/home/me/bin" {
		t.Fatalf("effect %+v", effect)
	}
	model, _ = Update(model, Listed{Dir: "/home/me/bin", Entries: nil})
	if model.Text != "/home/me/bin/out.json" || model.Selected != -1 {
		t.Fatalf("text %q selected %d", model.Text, model.Selected)
	}
	model, effect = Update(model, WentUp{})
	if effect.List != "/home/me" {
		t.Fatalf("effect %+v", effect)
	}
	model, _ = Update(model, Listed{Dir: "/home/me", Entries: home})
	if model.Text != "/home/me/out.json" {
		t.Fatalf("text %q", model.Text)
	}
}

func TestOpeningClearsTheFilterWhenTheFolderChanges(t *testing.T) {
	model, _ := Update(shown(t, Open, ""), Typed{Text: "/home/me/pro"})
	model, effect := Update(model, Entered{})
	if effect.List != "/home/me/Projects" {
		t.Fatalf("the selected match opens: %+v", effect)
	}
	model, _ = Update(model, Listed{Dir: "/home/me/Projects"})
	if model.Text != "/home/me/Projects/" {
		t.Fatalf("text %q", model.Text)
	}
}

func TestGoingUpStopsAtTheRoot(t *testing.T) {
	model, _ := New(Open, "/", "/root", "")
	model, _ = Update(model, Listed{Dir: "/", Entries: []workspace.FolderEntry{{Name: "etc", Dir: true}}})
	if model.Text != "/" {
		t.Fatalf("text %q", model.Text)
	}
	if _, effect := Update(model, WentUp{}); effect != (Effect{}) {
		t.Fatalf("there is nothing above the root: %+v", effect)
	}
}

func TestAtFolderTellsWhenBackspaceGoesUp(t *testing.T) {
	model := shown(t, Open, "")
	if !model.AtFolder() {
		t.Fatal("with nothing typed after the folder Backspace goes up")
	}
	model, _ = Update(model, Typed{Text: "/home/me/a"})
	if model.AtFolder() {
		t.Fatal("with a name typed Backspace deletes")
	}
}

func TestAFolderThatCannotBeListedKeepsTheOneShown(t *testing.T) {
	model := shown(t, Open, "")
	model, _ = Update(model, Moved{By: 1})
	model, _ = Update(model, Entered{})
	model, effect := Update(model, Listed{Dir: "/home/me/bin", Err: errors.New("permission denied")})
	if effect != (Effect{}) || model.Dir != "/home/me" || len(model.Visible()) != 5 {
		t.Fatalf("dir %q effect %+v", model.Dir, effect)
	}
	if model.Problem != "Cannot open /home/me/bin: permission denied" {
		t.Fatalf("problem %q", model.Problem)
	}
	model, _ = Update(model, Typed{Text: "/home/me/x"})
	if model.Problem != "" {
		t.Fatal("typing clears the problem")
	}
}

func TestStartFallsBackToHomeThenTheRoot(t *testing.T) {
	cases := []struct{ start, home, want string }{
		{"/work", "/home/me", "/work"},
		{"", "/home/me", "/home/me"},
		{"", "", "/"},
	}
	for _, tc := range cases {
		if _, effect := New(Open, tc.start, tc.home, ""); effect.List != tc.want {
			t.Errorf("start %q home %q: lists %q, want %q", tc.start, tc.home, effect.List, tc.want)
		}
	}
}

func TestCountSaysHowManyItemsAreShown(t *testing.T) {
	cases := []struct {
		typed string
		want  string
	}{{"/home/me/", "5 items"}, {"/home/me/notes", "1 item"}, {"/home/me/zzz", "No items"}}
	for _, tc := range cases {
		model, _ := Update(shown(t, Open, ""), Typed{Text: tc.typed})
		if got := model.Count(); got != tc.want {
			t.Errorf("typed %q: %q, want %q", tc.typed, got, tc.want)
		}
	}
}
