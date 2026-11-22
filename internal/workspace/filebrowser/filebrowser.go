// Package filebrowser is the state and the rules of the in-app file
// browser, the file chooser of a desktop that has no dialog program: which
// folder is shown, what the path field holds, which row is selected and
// what Enter does. It reads no disk: Update asks for a folder to be listed
// through its Effect, and the App answers with Listed.
package filebrowser

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
)

// Mode is what the browser asks for.
type Mode int

const (
	Open Mode = iota // an existing file
	Save             // a file to write, new or replaced
)

// maxName is the longest file name, in bytes, that Linux file systems take.
const maxName = 255

// Model is the browser's state.
type Model struct {
	Mode       Mode
	Dir        string             // the folder shown, absolute and clean; "" until the first listing
	ShowHidden bool               // names that start with a dot are listed
	Text       string             // the path field: the folder, a separator and the typed name
	Selected   int                // the selected row of Visible, -1 for none
	Problem    string             // why the last Enter did nothing, "" when it did
	Home       string             // what a typed ~ means
	entries    []port.FolderEntry // what Dir holds, folders first
	next       pending            // what the asked listing is for
}

// pending is what happens once an asked folder is listed.
type pending struct {
	name  string // the name the path field keeps after the folder
	enter bool   // press Enter again: the name was typed for that folder
}

// Effect is what the App does after an Update: list a folder and answer
// with Listed, or take the chosen path (asking first when Replace is set,
// since the file exists). The zero Effect asks nothing.
type Effect struct {
	List    string
	Chosen  string
	Replace bool
}

// New is a browser asking for a file in mode, in start (else home, else
// the root), with name as the save name. Its Effect lists the folder.
func New(mode Mode, start, home, name string) (Model, Effect) {
	dir := firstSet(start, home, string(filepath.Separator))
	model := Model{Mode: mode, Home: home, Selected: -1, next: pending{name: name}}
	return model, Effect{List: filepath.Clean(dir)}
}

func firstSet(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// Msg is something that happened to the browser.
type Msg interface{ browserMsg() }

// Listed answers an Effect's List: what dir holds, or why it could not be read.
type Listed struct {
	Dir     string
	Entries []port.FolderEntry
	Err     error
}

// Typed is the path field's new text.
type Typed struct{ Text string }

// Moved moves the selection by rows (Up is -1, Down 1).
type Moved struct{ By int }

// Picked selects a row by a click.
type Picked struct{ Index int }

// Entered is Enter or the Open and Save buttons.
type Entered struct{}

// WentUp shows the parent folder (Alt+Up, or Backspace at the folder).
type WentUp struct{}

// HiddenToggled shows or hides the names that start with a dot.
type HiddenToggled struct{}

func (Listed) browserMsg()        {}
func (Typed) browserMsg()         {}
func (Moved) browserMsg()         {}
func (Picked) browserMsg()        {}
func (Entered) browserMsg()       {}
func (WentUp) browserMsg()        {}
func (HiddenToggled) browserMsg() {}

// Update applies msg to model.
func Update(model Model, msg Msg) (Model, Effect) {
	switch msg := msg.(type) {
	case Listed:
		return model.listed(msg)
	case Typed:
		model.Text, model.Problem = msg.Text, ""
		model.Selected = model.typedSelection()
	case Moved:
		model.selectRow(model.Selected + msg.By)
	case Picked:
		model.selectRow(msg.Index)
	case Entered:
		return model.enter()
	case WentUp:
		return model.up()
	case HiddenToggled:
		model.ShowHidden, model.Selected = !model.ShowHidden, -1
	}
	return model, Effect{}
}

// listed shows a listed folder, or keeps the shown one when it could not be read.
func (model Model) listed(msg Listed) (Model, Effect) {
	next := model.next
	model.next = pending{}
	if msg.Err != nil {
		model.Problem = fmt.Sprintf("Cannot open %s: %v", msg.Dir, unwrapped(msg.Err))
		if model.Dir == "" { // the first folder failed: show it empty, so the field still works
			model.Dir, model.Text = msg.Dir, withName(msg.Dir, next.name)
		}
		return model, Effect{}
	}
	model.Dir, model.Problem, model.Selected = msg.Dir, "", -1
	model.entries = sorted(msg.Entries)
	model.Text = withName(msg.Dir, next.name)
	if next.enter {
		return model.enter()
	}
	return model, Effect{}
}

// unwrapped drops the adapter's "list <dir>:" context, which the problem
// already says.
func unwrapped(err error) error {
	for inner := errors.Unwrap(err); inner != nil; inner = errors.Unwrap(err) {
		err = inner
	}
	return err
}

// sorted is entries with folders first, then by name ignoring case.
func sorted(entries []port.FolderEntry) []port.FolderEntry {
	out := slices.Clone(entries)
	slices.SortStableFunc(out, func(left, right port.FolderEntry) int {
		if left.Dir != right.Dir {
			if left.Dir {
				return -1
			}
			return 1
		}
		if order := strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)); order != 0 {
			return order
		}
		return strings.Compare(left.Name, right.Name)
	})
	return out
}

// withName is the path field's text for name in dir.
func withName(dir, name string) string {
	sep := string(filepath.Separator)
	return strings.TrimSuffix(dir, sep) + sep + name
}

// split cuts the path field into the folder it names and the name after it.
func (model Model) split() (dir, name string) {
	cut := strings.LastIndex(model.Text, string(filepath.Separator))
	if cut < 0 {
		return model.Dir, model.Text
	}
	return model.resolve(model.Text[:cut+1]), model.Text[cut+1:]
}

// resolve makes a typed folder absolute: ~ is home, a relative one is under Dir.
func (model Model) resolve(typed string) string {
	sep := string(filepath.Separator)
	if model.Home != "" && (typed == "~" || strings.HasPrefix(typed, "~"+sep)) {
		typed = model.Home + typed[1:]
	}
	if !filepath.IsAbs(typed) {
		typed = filepath.Join(model.Dir, typed)
	}
	return filepath.Clean(typed)
}

// filter is the typed name the list is narrowed to: only while opening and
// only in the shown folder, since a save name is not a search.
func (model Model) filter() string {
	dir, name := model.split()
	if model.Mode != Open || dir != model.Dir {
		return ""
	}
	return name
}

// Visible are the rows shown: the folder's entries without hidden ones
// (unless shown, or a dot is typed) and narrowed to the typed name, a
// part of it or a pattern such as *.json, ignoring case.
func (model Model) Visible() []port.FolderEntry {
	filter := strings.ToLower(model.filter())
	hidden := model.ShowHidden || strings.HasPrefix(filter, ".")
	var out []port.FolderEntry
	for _, entry := range model.entries {
		if !hidden && strings.HasPrefix(entry.Name, ".") {
			continue
		}
		if matches(filter, strings.ToLower(entry.Name)) {
			out = append(out, entry)
		}
	}
	return out
}

func matches(filter, name string) bool {
	if strings.ContainsAny(filter, "*?[") {
		ok, err := filepath.Match(filter, name)
		return err == nil && ok
	}
	return strings.Contains(name, filter)
}

// typedSelection is the row Enter takes after typing: while opening the
// exact name, else the first match; while saving none, since the typed
// name is what is saved.
func (model Model) typedSelection() int {
	filter := model.filter()
	visible := model.Visible()
	if filter == "" || len(visible) == 0 {
		return -1
	}
	for i, entry := range visible {
		if strings.EqualFold(entry.Name, filter) {
			return i
		}
	}
	return 0
}

// selectRow selects row (kept inside the list); while saving a file's
// name becomes the save name.
func (model *Model) selectRow(row int) {
	visible := model.Visible()
	if len(visible) == 0 {
		model.Selected = -1
		return
	}
	model.Selected = min(max(row, 0), len(visible)-1)
	if entry := visible[model.Selected]; model.Mode == Save && !entry.Dir {
		model.Text = withName(model.Dir, entry.Name)
	}
}

// AtFolder reports whether nothing is typed after the folder, when
// Backspace goes up instead of deleting.
func (model Model) AtFolder() bool {
	_, name := model.split()
	return name == "" && strings.HasSuffix(model.Text, string(filepath.Separator))
}

// Count says how many rows are shown, for the dialog's subtitle.
func (model Model) Count() string {
	switch count := len(model.Visible()); count {
	case 0:
		return "No items"
	case 1:
		return "1 item"
	default:
		return fmt.Sprintf("%d items", count)
	}
}

// enter does what Enter does: the selected row first, else the typed path.
func (model Model) enter() (Model, Effect) {
	model.Problem = ""
	if visible := model.Visible(); model.Selected >= 0 && model.Selected < len(visible) {
		return model.take(visible[model.Selected])
	}
	dir, name := model.split()
	if dir != model.Dir {
		return model.list(dir, pending{name: name, enter: name != ""})
	}
	switch name {
	case "", ".":
		if model.Mode == Save {
			model.Problem = "Type a name for the file"
		}
		return model, Effect{}
	case "..":
		return model.list(filepath.Dir(model.Dir), pending{})
	}
	if entry, ok := model.lookup(name); ok {
		return model.take(entry)
	}
	if model.Mode == Open {
		model.Problem = "There is no " + name + " here"
		return model, Effect{}
	}
	if problem := nameProblem(name); problem != "" {
		model.Problem = problem
		return model, Effect{}
	}
	return model, Effect{Chosen: filepath.Join(model.Dir, name)}
}

// take opens a folder or chooses a file; an existing file is replaced only
// after asking.
func (model Model) take(entry port.FolderEntry) (Model, Effect) {
	path := filepath.Join(model.Dir, entry.Name)
	if entry.Dir {
		return model.list(path, pending{name: model.keptName(entry.Name)})
	}
	return model, Effect{Chosen: path, Replace: model.Mode == Save}
}

// keptName is the name the path field keeps when the folder changes:
// while saving the save name, unless it named the folder entered.
func (model Model) keptName(entered string) string {
	_, name := model.split()
	if model.Mode != Save || name == entered {
		return ""
	}
	return name
}

// up lists the parent folder; the root has none.
func (model Model) up() (Model, Effect) {
	parent := filepath.Dir(model.Dir)
	if model.Dir == "" || parent == model.Dir {
		return model, Effect{}
	}
	return model.list(parent, pending{name: model.keptName("")})
}

func (model Model) list(dir string, next pending) (Model, Effect) {
	model.next = next
	return model, Effect{List: dir}
}

// lookup finds an entry by its exact name, hidden or not.
func (model Model) lookup(name string) (port.FolderEntry, bool) {
	for _, entry := range model.entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return port.FolderEntry{}, false
}

// nameProblem is why name cannot be a file name, "" when it can.
func nameProblem(name string) string {
	switch {
	case len(name) > maxName:
		return fmt.Sprintf("The name is longer than %d bytes", maxName)
	case strings.ContainsRune(name, 0):
		return "A file name cannot hold a NUL character"
	}
	return ""
}
