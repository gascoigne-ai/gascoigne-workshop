package collections

import (
	"errors"
	"fmt"
	"slices"

	collection "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// ErrNotOpen says the collection an operation names is not in the open set.
var ErrNotOpen = errors.New("the collection is not open")

// Origin says where the open set came from, which decides how it is
// written back to the settings. It is a two-value enum and not a bool
// ("is remembered"), so the two ways of writing are named where they are
// chosen; every change by the user moves a set to OriginRemembered and it
// never goes back.
type Origin int

const (
	// OriginStartFolder is a set opened from a folder given at start; only
	// the entries of the remembered list that name an open collection change.
	OriginStartFolder Origin = iota
	// OriginRemembered is a set opened from the remembered list or changed
	// by the user since; it is written whole.
	OriginRemembered
)

// Settings is where the remembered list of the open set is kept: the
// settings feature, which this controller may not import. Remember writes
// the whole list.
type Settings interface {
	Remembered() []request.OpenCollection
	Remember(list []request.OpenCollection)
}

// Saves is the autosave bookkeeping of the open collections, which the
// workspace feature (autosave.Collections) owns and this controller may not
// import; the App answers it.
// Saved marks col as written as it is now; FlushAll writes every collection
// that changed.
type Saves interface {
	Saved(col *request.Collection)
	FlushAll() error
}

// Deps is what Library needs from features it may not import. Nothing in it
// raises a prompt or a toast: those stay in the window.
type Deps struct {
	Settings Settings
	Saves    Saves
}

// Opening is what Open did: the collection, the request to show in it and
// whether it was read now (the caller then opens its folders in the tree)
// or was open already.
type Opening struct {
	Collection *request.Collection
	Show       *request.Request
	Loaded     bool
}

// Failure is a remembered folder Restore could not open and why.
type Failure struct {
	Folder string
	Err    error
}

// Library is the open set with the operations that change it: open, create,
// rename, delete, close, move and locate, each run through the Store port.
// It owns the Set (there is no second copy) and the remembered-list
// bookkeeping, and is used from the UI goroutine only.
//
// Why a type holding the port and not functions taking it: every operation
// needs the same store, the collections folder and the same three pieces of
// state (set, origin, unreadable), and a caller that passed them in would
// have to keep them in step. Rejected: leaving the operations in the App,
// where a test needs a whole window to run one; revisit when a second
// consumer needs a different persistence of the open set.
//
// Prompts, confirmations and toasts are not here. An operation returns an
// error worded for the user in lower case (the App upper-cases its first
// letter and shows it), so the text a user reads is decided where the
// failure is known and shown where the window is. Saving a collection to its
// folder in the background is the workspace feature's (autosave.Collections,
// built over the same Store); this controller only asks it, through Saves,
// to mark a collection written and to flush.
type Library struct {
	store      collection.Store
	deps       Deps
	parent     string // folder new collections are made in, "" when there is none
	set        Set
	origin     Origin
	unreadable []request.OpenCollection // remembered folders that failed to load; kept in the list so a mount or a fix brings them back
}

// NewLibrary returns a library with nothing open. parent is the folder new
// collections are made in, beside the default workspace and never inside
// it (a folder with a folder.json below a collection is one of its
// folders); "" when the program has no config directory.
func NewLibrary(store collection.Store, parent string, deps Deps) *Library {
	return &Library{store: store, deps: deps, parent: parent}
}

// Set is the open collections, in tree order.
func (lib *Library) Set() Set { return lib.set }

// Apply changes the open set by msg, as Update does. It writes nothing to
// the settings: the operations that should do so remember themselves.
func (lib *Library) Apply(msg Msg) { lib.set = Update(lib.set, msg) }

// Origin is where the open set came from.
func (lib *Library) Origin() Origin { return lib.origin }

// Unreadable are the remembered folders that could not be opened and are
// kept in the list.
func (lib *Library) Unreadable() []request.OpenCollection { return lib.unreadable }

// FirstOrBlank is the request to show for col: its first request, or a new
// blank one (REST, GET) in its root when it has none. The blank request
// counts as an edit of col, so a caller that wants it saved marks col saved
// afterwards.
func FirstOrBlank(col *request.Collection) *request.Request {
	if first := tree.FirstRequest(&col.Folder); first != nil {
		return first
	}
	return col.AddRequest(request.Request{Name: "New Request", Protocol: request.ProtocolREST, Method: "GET"})
}

// Open opens the collection saved in dir after the open ones, with the
// environment remembered for the folder. A folder already open changes
// nothing (its edits stay pending and the remembered list stays as it is,
// since the set did not change); one that cannot be read is not opened.
func (lib *Library) Open(dir string) (Opening, error) {
	dir = AbsDir(dir)
	if entry, ok := lib.set.ByDir(dir); ok {
		return Opening{Collection: entry.Collection, Show: FirstOrBlank(entry.Collection)}, nil
	}
	loaded, err := lib.store.Load(dir)
	if err != nil {
		return Opening{}, err
	}
	col := &loaded
	lib.include(col, dir, lib.RememberedEnv(dir))
	return Opening{Collection: col, Show: lib.shown(col), Loaded: true}, nil
}

// Reopen replaces the open set with the collection loaded from dir, with
// the environment called env active, and returns the request to show. The
// caller has saved and torn down what was open.
func (lib *Library) Reopen(loaded request.Collection, dir, env string) *request.Request {
	lib.set = Set{}
	col := &loaded
	lib.include(col, dir, env)
	return lib.shown(col)
}

// shown is FirstOrBlank for a collection just opened: the blank request a
// new collection gets is not an edit, and the set is remembered.
func (lib *Library) shown(col *request.Collection) *request.Request {
	show := FirstOrBlank(col)
	lib.deps.Saves.Saved(col)
	lib.remember()
	return show
}

// include puts col, saved in dir ("" for not saved), into the open set with
// the environment called env active, and counts it as saved.
func (lib *Library) include(col *request.Collection, dir, env string) {
	lib.Apply(Open{Collection: col, Dir: dir})
	lib.Apply(ChooseEnv{Collection: col, Index: col.EnvIndex(env)})
	lib.deps.Saves.Saved(col)
}

// Restore opens the collections of wanted in order, the way the program
// does at start. origin says where the list came from. A folder that cannot
// be read is left out and returned; a list from the settings keeps it too, so
// a later write does not lose it. An entry without a folder stands for the
// collection the program starts with: start makes it (once, and only when
// none unsaved is open yet). When nothing is open at the end, start makes
// the one for the environment remembered for "". A nil start is a build that
// starts with no collection: the entries without a folder are skipped and the
// set may stay empty, as it is after the last collection is closed.
func (lib *Library) Restore(wanted []request.OpenCollection, origin Origin, start func(env string) *request.Collection) []Failure {
	lib.origin, lib.unreadable = origin, nil
	var failures []Failure
	for _, item := range wanted {
		if item.Folder == "" {
			lib.includeStart(item.Environment, start)
			continue
		}
		if err := lib.restoreFolder(item); err != nil {
			failures = append(failures, Failure{Folder: item.Folder, Err: err})
			if origin == OriginRemembered {
				lib.unreadable = append(lib.unreadable, item)
			}
		}
	}
	if len(lib.set.Entries) == 0 {
		lib.includeStart(lib.RememberedEnv(""), start)
	}
	return failures
}

func (lib *Library) restoreFolder(item request.OpenCollection) error {
	dir := AbsDir(item.Folder)
	if _, open := lib.set.ByDir(dir); open {
		return nil
	}
	loaded, err := lib.store.Load(dir)
	if err != nil {
		return err
	}
	lib.include(&loaded, dir, item.Environment)
	return nil
}

func (lib *Library) includeStart(env string, start func(env string) *request.Collection) {
	if start == nil {
		return
	}
	if _, open := lib.set.At(""); open {
		return
	}
	lib.include(start(env), "", env)
}

// Create makes a new empty collection called text in the collections folder
// and returns its folder; opening it is the caller's next step.
func (lib *Library) Create(text string) (dir string, err error) {
	name, err := CleanName(text)
	if err != nil {
		return "", fmt.Errorf("cannot create a collection: %w", err)
	}
	if lib.parent == "" {
		return "", errors.New("there is no folder for new collections")
	}
	dir, err = lib.store.Create(lib.parent, name)
	if err != nil {
		return "", fmt.Errorf("cannot create the collection: %w", err)
	}
	return dir, nil
}

// Rename gives col the name text and moves its folder to match. The writes
// in flight go to the old folder first.
func (lib *Library) Rename(col *request.Collection, text string) (name string, err error) {
	name, err = CleanName(text)
	if err != nil {
		return "", fmt.Errorf("cannot rename: %w", err)
	}
	if err := lib.deps.Saves.FlushAll(); err != nil {
		return "", fmt.Errorf("saving the collection failed: %w", err)
	}
	entry, ok := lib.set.ByCollection(col)
	if !ok {
		return "", ErrNotOpen
	}
	dir, err := lib.store.Rename(entry.Dir, name)
	if err != nil {
		return "", fmt.Errorf("cannot rename: %w", err)
	}
	col.Name = name
	col.MoveProtoPaths(entry.Dir, dir)
	lib.Apply(Located{Collection: col, Dir: dir})
	lib.remember()
	lib.deps.Saves.Saved(col)
	return name, nil
}

// Delete removes col from the disk. The caller commits its edits and waits
// for the writes in flight first. next is the folder to open in its place
// when it was the only collection open: the next of the collections folder,
// or a new empty one when none is left; "" when others stay open, and then
// the caller closes col.
func (lib *Library) Delete(col *request.Collection) (next string, err error) {
	entry, ok := lib.set.ByCollection(col)
	if !ok {
		return "", ErrNotOpen
	}
	if err := lib.store.Delete(entry.Dir); err != nil {
		return "", fmt.Errorf("cannot delete the collection: %w", err)
	}
	lib.Apply(Located{Collection: col, Dir: ""}) // nothing is left to flush into the deleted folder
	if len(lib.set.Entries) > 1 {
		return "", nil
	}
	listed, _ := lib.store.List(lib.parent) // a folder that cannot be listed has no next collection
	next = NextAfterDelete(listed, entry.Dir)
	if next != "" {
		return next, nil
	}
	if next, err = lib.store.Create(lib.parent, "My collection"); err != nil {
		return "", fmt.Errorf("deleted, but no collection could be opened: %w", err)
	}
	return next, nil
}

// Locate makes the folder of an unsaved collection, named after it, in the
// collections folder, and records it.
func (lib *Library) Locate(col *request.Collection) (dir string, err error) {
	if lib.parent == "" {
		return "", errors.New("no folder to save to")
	}
	name, err := CleanName(col.Name)
	if err != nil {
		name = "My collection" // the folder needs a name even when the collection's cannot be one
	}
	dir, err = lib.store.Create(lib.parent, name)
	if err != nil {
		return "", fmt.Errorf("saving failed: %w", err)
	}
	lib.Apply(Located{Collection: col, Dir: dir})
	lib.remember()
	return dir, nil
}

// Close takes col out of the open set and remembers the set left. The App
// tears down the tabs of col first.
func (lib *Library) Close(col *request.Collection) {
	lib.Apply(Close{Collection: col})
	lib.remember()
}

// Shift moves col one place up the tree (by -1) or down (by 1) and reports
// whether it moved; at an end it stays and nothing is remembered.
func (lib *Library) Shift(col *request.Collection, by int) bool {
	moved := Update(lib.set, Shift{Collection: col, By: by})
	if slices.Equal(moved.Entries, lib.set.Entries) {
		return false
	}
	lib.set = moved
	lib.remember()
	return true
}

// AsksToClose reports whether closing col would lose edits, so the user is
// asked first: a collection with a folder is written as it closes, but one
// without a folder keeps its edits nowhere. pending says col changed since
// it was last written.
func (lib *Library) AsksToClose(col *request.Collection, pending bool) bool {
	entry, ok := lib.set.ByCollection(col)
	return ok && entry.Dir == "" && pending
}

// remember writes the open set to the settings, which then no longer carry
// the older single workspace and environment.
func (lib *Library) remember() {
	lib.deps.Settings.Remember(Remembered(lib.set, lib.unreadable))
	lib.origin = OriginRemembered
}

// RememberEnvironments writes the active environment of each open
// collection to the settings. When the open set did not come from the
// remembered list (a folder given at start with -workspace), only the
// entries of the list that name an open collection change: the list is what
// the next ordinary start opens and such a run must not replace it.
func (lib *Library) RememberEnvironments() {
	if lib.origin == OriginRemembered {
		lib.remember()
		return
	}
	remembered, changed := WithEnvironments(lib.set, lib.deps.Settings.Remembered())
	if changed {
		lib.deps.Settings.Remember(remembered)
	}
}

// RememberedEnv is the environment remembered for the collection in dir
// (absolute), "" for none.
func (lib *Library) RememberedEnv(dir string) string {
	for _, item := range lib.deps.Settings.Remembered() {
		if item.Folder == dir || (item.Folder != "" && AbsDir(item.Folder) == dir) {
			return item.Environment
		}
	}
	return ""
}
