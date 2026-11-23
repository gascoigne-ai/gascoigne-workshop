package collections

import (
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// fakeStore keeps collections by folder and logs each call, so a test sees
// what the library asked the port and in which order.
type fakeStore struct {
	folders map[string]request.Collection
	calls   []string
	fail    map[string]error // by method name
}

func newFakeStore() *fakeStore {
	return &fakeStore{folders: map[string]request.Collection{}, fail: map[string]error{}}
}

func (store *fakeStore) call(method, arg string) error {
	store.calls = append(store.calls, method+" "+arg)
	return store.fail[method]
}

func (store *fakeStore) called(method string) bool {
	return slices.ContainsFunc(store.calls, func(call string) bool { return strings.HasPrefix(call, method+" ") })
}

func (store *fakeStore) Load(dir string) (request.Collection, error) {
	if err := store.call("Load", dir); err != nil {
		return request.Collection{}, err
	}
	col, ok := store.folders[dir]
	if !ok {
		return request.Collection{}, errors.New("no collection in " + dir)
	}
	return col, nil
}

func (store *fakeStore) Save(dir string, col request.Collection) error {
	if err := store.call("Save", dir); err != nil {
		return err
	}
	store.folders[dir] = col
	return nil
}

func (store *fakeStore) Create(parent, name string) (string, error) {
	if err := store.call("Create", parent+" "+name); err != nil {
		return "", err
	}
	dir := filepath.Join(parent, name)
	store.folders[dir] = request.Collection{Folder: request.Folder{Name: name}}
	return dir, nil
}

func (store *fakeStore) Rename(dir, name string) (string, error) {
	if err := store.call("Rename", dir+" "+name); err != nil {
		return "", err
	}
	renamed := filepath.Join(filepath.Dir(dir), name)
	store.folders[renamed] = store.folders[dir]
	delete(store.folders, dir)
	return renamed, nil
}

func (store *fakeStore) Delete(dir string) error {
	if err := store.call("Delete", dir); err != nil {
		return err
	}
	delete(store.folders, dir)
	return nil
}

func (store *fakeStore) List(parent string) ([]string, error) {
	if err := store.call("List", parent); err != nil {
		return nil, err
	}
	var dirs []string
	for dir := range store.folders {
		if filepath.Dir(dir) == parent {
			dirs = append(dirs, dir)
		}
	}
	slices.Sort(dirs)
	return dirs, nil
}

type fakeSettings struct {
	list      []request.OpenCollection
	remembers [][]request.OpenCollection
}

func (settings *fakeSettings) Remembered() []request.OpenCollection { return settings.list }
func (settings *fakeSettings) Remember(list []request.OpenCollection) {
	settings.list = list
	settings.remembers = append(settings.remembers, list)
}

type fakeSaves struct {
	saved    []*request.Collection
	flushErr error
	flushes  int
}

func (saves *fakeSaves) Saved(col *request.Collection) { saves.saved = append(saves.saved, col) }
func (saves *fakeSaves) FlushAll() error {
	saves.flushes++
	return saves.flushErr
}

// library is a Library over fakes, with the collections folder "/c".
func library() (*Library, *fakeStore, *fakeSettings, *fakeSaves) {
	store, settings, saves := newFakeStore(), &fakeSettings{}, &fakeSaves{}
	return NewLibrary(store, "/c", Deps{Settings: settings, Saves: saves}), store, settings, saves
}

// stored puts a collection with one request and the environments dev and
// prod into the fake store at dir.
func (store *fakeStore) stored(dir string) {
	col := request.Collection{Folder: request.Folder{Name: filepath.Base(dir)}}
	col.Envs = []request.Environment{{Name: "dev"}, {Name: "prod"}}
	col.AddRequest(request.Request{Name: "first"})
	store.folders[dir] = col
}

func TestFirstOrBlankShowsTheFirstRequestOrMakesABlankOne(t *testing.T) {
	withRequest, first := holding("a")
	if got := FirstOrBlank(withRequest); got != first {
		t.Errorf("got %v, want the first request", got)
	}
	empty := newCollection("b")
	got := FirstOrBlank(empty)
	if got == nil || got.Name != "New Request" || got.Protocol != request.ProtocolREST || got.Method != "GET" {
		t.Fatalf("got %+v, want a blank GET request", got)
	}
	if len(empty.Requests) != 1 || empty.Requests[0] != got {
		t.Error("the blank request is not in the collection root")
	}
}

func TestOpenAnOpenFolderChangesNothing(t *testing.T) {
	lib, store, settings, saves := library()
	store.stored("/w/a")
	if _, err := lib.Open("/w/a"); err != nil {
		t.Fatal(err)
	}
	store.calls, settings.remembers, saves.saved = nil, nil, nil
	opening, err := lib.Open("/w/a")
	if err != nil || opening.Loaded || opening.Collection == nil || opening.Show == nil {
		t.Fatalf("opening %+v, err %v; want the open collection, not loaded", opening, err)
	}
	if store.called("Load") || len(settings.remembers) != 0 || len(saves.saved) != 0 || len(lib.Set().Entries) != 1 {
		t.Errorf("loads %v, remembers %d, saved %d, open %d; want nothing done", store.calls, len(settings.remembers), len(saves.saved), len(lib.Set().Entries))
	}
}

func TestOpenAFreshFolderUsesTheRememberedEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		list     []request.OpenCollection
		wantEnv  int
		wantName string
	}{
		{"remembered", []request.OpenCollection{{Folder: "/w/a", Environment: "prod"}}, 1, "prod"},
		{"unknown name", []request.OpenCollection{{Folder: "/w/a", Environment: "gone"}}, -1, ""},
		{"not remembered", nil, -1, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lib, store, settings, saves := library()
			store.stored("/w/a")
			settings.list = test.list
			opening, err := lib.Open("/w/a")
			if err != nil || !opening.Loaded {
				t.Fatalf("opening %+v, err %v", opening, err)
			}
			entry, ok := lib.Set().ByCollection(opening.Collection)
			if !ok || entry.Dir != "/w/a" || entry.Env != test.wantEnv {
				t.Errorf("entry %+v (open %v), want folder /w/a, environment %d", entry, ok, test.wantEnv)
			}
			if len(settings.remembers) != 1 {
				t.Errorf("%d writes of the remembered list, want 1", len(settings.remembers))
			}
			if len(saves.saved) != 2 {
				t.Errorf("%d saved marks, want 2 (opened, then after the blank request)", len(saves.saved))
			}
			if opening.Show == nil || opening.Show.Name != "first" {
				t.Errorf("show %+v, want the first request", opening.Show)
			}
		})
	}
}

func TestOpenMakesARelativeFolderAbsolute(t *testing.T) {
	lib, store, _, _ := library()
	abs := AbsDir("rel/col")
	store.stored(abs)
	opening, err := lib.Open("rel/col")
	if err != nil || !opening.Loaded {
		t.Fatalf("opening %+v, err %v", opening, err)
	}
	if entry, _ := lib.Set().ByCollection(opening.Collection); entry.Dir != abs {
		t.Errorf("folder %q, want %q", entry.Dir, abs)
	}
}

func TestOpenThatCannotLoadLeavesTheSetAlone(t *testing.T) {
	lib, store, settings, _ := library()
	store.fail["Load"] = errors.New("unreadable")
	if _, err := lib.Open("/w/a"); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("err = %v, want the load error", err)
	}
	if len(lib.Set().Entries) != 0 || len(settings.remembers) != 0 {
		t.Errorf("open %d, remembers %d; want none", len(lib.Set().Entries), len(settings.remembers))
	}
}

func TestCreateRefusesAndNamesTheReason(t *testing.T) {
	tests := []struct {
		name    string
		parent  string
		text    string
		failure error
		want    string
	}{
		{"bad name", "/c", "  ", nil, "cannot create a collection: the name is empty"},
		{"no folder for collections", "", "Alpha", nil, "there is no folder for new collections"},
		{"the store fails", "/c", "Alpha", errors.New("disk full"), "cannot create the collection: disk full"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, store, settings, saves := library()
			lib := NewLibrary(store, test.parent, Deps{Settings: settings, Saves: saves})
			store.fail["Create"] = test.failure
			dir, err := lib.Create(test.text)
			if err == nil || err.Error() != test.want || dir != "" {
				t.Errorf("dir %q, err %v; want %q", dir, err, test.want)
			}
		})
	}
}

func TestCreateMakesTheFolderBelowTheCollectionsFolder(t *testing.T) {
	lib, store, _, _ := library()
	dir, err := lib.Create(" Alpha ")
	if err != nil || dir != "/c/Alpha" || store.calls[0] != "Create /c Alpha" {
		t.Fatalf("dir %q, err %v, calls %v", dir, err, store.calls)
	}
}

// openTwo opens /w/a and /w/b and clears the logs.
func openTwo(t *testing.T) (*Library, *fakeStore, *fakeSettings, *fakeSaves, *request.Collection, *request.Collection) {
	t.Helper()
	lib, store, settings, saves := library()
	store.stored("/w/a")
	store.stored("/w/b")
	first, err := lib.Open("/w/a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := lib.Open("/w/b")
	if err != nil {
		t.Fatal(err)
	}
	store.calls, settings.remembers, saves.saved = nil, nil, nil
	return lib, store, settings, saves, first.Collection, second.Collection
}

func TestRenameRefusesBeforeTouchingAnything(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		flushErr   error
		storeErr   error
		want       string
		wantFlush  int
		wantRename bool
	}{
		{"bad name", " ", nil, nil, "cannot rename: the name is empty", 0, false},
		{"flush fails", "Beta", errors.New("busy"), nil, "saving the collection failed: busy", 1, false},
		{"store fails", "Beta", nil, errors.New("exists"), "cannot rename: exists", 1, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lib, store, settings, saves, first, _ := openTwo(t)
			saves.flushErr, store.fail["Rename"] = test.flushErr, test.storeErr
			name, err := lib.Rename(first, test.text)
			if err == nil || err.Error() != test.want || name != "" {
				t.Errorf("name %q, err %v; want %q", name, err, test.want)
			}
			if saves.flushes != test.wantFlush || store.called("Rename") != test.wantRename {
				t.Errorf("flushes %d, rename called %v", saves.flushes, store.called("Rename"))
			}
			if entry, _ := lib.Set().ByCollection(first); entry.Dir != "/w/a" || first.Name == "Beta" || len(settings.remembers) != 0 {
				t.Errorf("folder %q, name %q, remembers %d; want nothing changed", entry.Dir, first.Name, len(settings.remembers))
			}
		})
	}
}

func TestRenameMovesTheCollectionAndItsProtoPaths(t *testing.T) {
	lib, _, settings, saves, first, _ := openTwo(t)
	first.AddRequest(request.Request{Name: "grpc", Protocol: request.ProtocolGRPC, Proto: request.ProtoSource{Files: []string{"/w/a/api.proto"}}})
	name, err := lib.Rename(first, "Beta")
	if err != nil || name != "Beta" || first.Name != "Beta" {
		t.Fatalf("name %q, err %v, collection %q", name, err, first.Name)
	}
	if entry, _ := lib.Set().ByCollection(first); entry.Dir != "/w/Beta" {
		t.Errorf("folder %q, want /w/Beta", entry.Dir)
	}
	if got := first.Requests[len(first.Requests)-1].Proto.Files; !reflect.DeepEqual(got, []string{"/w/Beta/api.proto"}) {
		t.Errorf("proto files %v, want them moved with the folder", got)
	}
	if len(settings.remembers) != 1 || settings.list[0].Folder != "/w/Beta" {
		t.Errorf("remembered %+v", settings.list)
	}
	if len(saves.saved) != 1 || saves.saved[0] != first {
		t.Errorf("saved marks %v, want the renamed collection once", saves.saved)
	}
}

func TestRenameOfAClosedCollectionIsNotOpen(t *testing.T) {
	lib, _, _, _ := library()
	_, err := lib.Rename(newCollection("x"), "Beta")
	if !errors.Is(err, ErrNotOpen) {
		t.Errorf("err = %v, want ErrNotOpen", err)
	}
}

func TestDeleteWithOthersOpenLeavesNoNext(t *testing.T) {
	lib, store, _, _, first, _ := openTwo(t)
	next, err := lib.Delete(first)
	if err != nil || next != "" {
		t.Fatalf("next %q, err %v", next, err)
	}
	if entry, _ := lib.Set().ByCollection(first); entry.Dir != "" {
		t.Errorf("folder %q, want it cleared so nothing is flushed into the deleted folder", entry.Dir)
	}
	if !slices.Contains(store.calls, "Delete /w/a") {
		t.Errorf("calls %v", store.calls)
	}
}

func TestDeleteOfTheLastOneNamesWhatOpensNext(t *testing.T) {
	tests := []struct {
		name   string
		listed []string
		want   string
		made   bool
	}{
		{"another is listed", []string{"/c/other"}, "/c/other", false},
		{"none is listed", nil, "/c/My collection", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lib, store, _, _ := library()
			store.stored("/c/gone")
			for _, dir := range test.listed {
				store.stored(dir)
			}
			opening, err := lib.Open("/c/gone")
			if err != nil {
				t.Fatal(err)
			}
			next, err := lib.Delete(opening.Collection)
			if err != nil || next != test.want || store.called("Create") != test.made {
				t.Errorf("next %q, err %v, created %v; want %q, created %v", next, err, store.called("Create"), test.want, test.made)
			}
		})
	}
}

func TestDeleteFailuresSayWhatHappened(t *testing.T) {
	t.Run("the store cannot delete", func(t *testing.T) {
		lib, store, _, _, first, _ := openTwo(t)
		store.fail["Delete"] = errors.New("busy")
		_, err := lib.Delete(first)
		if err == nil || err.Error() != "cannot delete the collection: busy" {
			t.Fatalf("err = %v", err)
		}
		if entry, _ := lib.Set().ByCollection(first); entry.Dir != "/w/a" {
			t.Errorf("folder %q, want the set unchanged", entry.Dir)
		}
	})
	t.Run("nothing can be opened after the last", func(t *testing.T) {
		lib, store, _, _ := library()
		store.stored("/c/gone")
		opening, _ := lib.Open("/c/gone")
		store.fail["Create"] = errors.New("read-only")
		_, err := lib.Delete(opening.Collection)
		if err == nil || err.Error() != "deleted, but no collection could be opened: read-only" {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("a closed collection", func(t *testing.T) {
		lib, _, _, _ := library()
		if _, err := lib.Delete(newCollection("x")); !errors.Is(err, ErrNotOpen) {
			t.Errorf("err = %v, want ErrNotOpen", err)
		}
	})
}

func TestCloseRemovesAndRemembersAndShiftRemembersOnlyWhenMoved(t *testing.T) {
	lib, _, settings, _, first, second := openTwo(t)
	if lib.Shift(first, -1) || len(settings.remembers) != 0 {
		t.Error("shifting the first up moved it or wrote the list")
	}
	if !lib.Shift(first, 1) || lib.Set().Index(first) != 1 || len(settings.remembers) != 1 {
		t.Errorf("shifting down: index %d, writes %d", lib.Set().Index(first), len(settings.remembers))
	}
	lib.Close(second)
	if len(lib.Set().Entries) != 1 || len(settings.remembers) != 2 {
		t.Errorf("open %d, writes %d; want 1 open and the list written", len(lib.Set().Entries), len(settings.remembers))
	}
}

func TestLocateMakesTheFolderOfAnUnsavedCollection(t *testing.T) {
	t.Run("no folder for collections", func(t *testing.T) {
		_, store, settings, saves := library()
		lib := NewLibrary(store, "", Deps{Settings: settings, Saves: saves})
		if _, err := lib.Locate(newCollection("x")); err == nil || err.Error() != "no folder to save to" {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("the store fails", func(t *testing.T) {
		lib, store, _, _ := library()
		store.fail["Create"] = errors.New("denied")
		if _, err := lib.Locate(newCollection("x")); err == nil || err.Error() != "saving failed: denied" {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("success", func(t *testing.T) {
		lib, _, settings, _ := library()
		col := newCollection("Draft")
		lib.Apply(Open{Collection: col})
		dir, err := lib.Locate(col)
		if err != nil || dir != "/c/Draft" {
			t.Fatalf("dir %q, err %v", dir, err)
		}
		if entry, _ := lib.Set().ByCollection(col); entry.Dir != dir || len(settings.remembers) != 1 {
			t.Errorf("folder %q, writes %d", entry.Dir, len(settings.remembers))
		}
	})
	t.Run("a name that cannot name a folder", func(t *testing.T) {
		lib, _, _, _ := library()
		col := newCollection("")
		lib.Apply(Open{Collection: col})
		if dir, err := lib.Locate(col); err != nil || dir != "/c/My collection" {
			t.Errorf("dir %q, err %v", dir, err)
		}
	})
}

func TestAsksToCloseOnlyWhenEditsWouldBeLost(t *testing.T) {
	lib, _, _, _ := library()
	saved, unsaved := newCollection("saved"), newCollection("unsaved")
	lib.Apply(Open{Collection: saved, Dir: "/w/saved"})
	lib.Apply(Open{Collection: unsaved})
	tests := []struct {
		name    string
		col     *request.Collection
		pending bool
		want    bool
	}{
		{"a folder with pending edits", saved, true, false},
		{"no folder with pending edits", unsaved, true, true},
		{"no folder, nothing pending", unsaved, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := lib.AsksToClose(test.col, test.pending); got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestRestoreFromTheRememberedListKeepsUnreadableFolders(t *testing.T) {
	lib, store, settings, _ := library()
	store.stored("/w/a")
	started := 0
	wanted := []request.OpenCollection{{Folder: "/w/a", Environment: "dev"}, {Folder: "/w/gone"}}
	failures := lib.Restore(wanted, OriginRemembered, func(string) *request.Collection { started++; return newCollection("s") })
	if len(failures) != 1 || failures[0].Folder != "/w/gone" || failures[0].Err == nil {
		t.Fatalf("failures %+v", failures)
	}
	if started != 0 || len(lib.Set().Entries) != 1 || lib.Origin() != OriginRemembered {
		t.Errorf("started %d, open %d, origin %v", started, len(lib.Set().Entries), lib.Origin())
	}
	lib.RememberEnvironments()
	want := []request.OpenCollection{{Folder: "/w/a", Environment: "dev"}, {Folder: "/w/gone"}}
	if !reflect.DeepEqual(settings.list, want) {
		t.Errorf("remembered %+v, want the unreadable folder kept: %+v", settings.list, want)
	}
}

func TestRestoreFromAStartFolderDoesNotRecordUnreadable(t *testing.T) {
	lib, store, settings, _ := library()
	store.stored("/w/a")
	settings.list = []request.OpenCollection{{Folder: "/w/a", Environment: "prod"}, {Folder: "/w/other", Environment: "dev"}}
	lib.Restore([]request.OpenCollection{{Folder: "/w/gone"}, {Folder: "/w/a", Environment: "dev"}}, OriginStartFolder, func(string) *request.Collection {
		return newCollection("s")
	})
	if len(lib.Unreadable()) != 0 || lib.Origin() != OriginStartFolder {
		t.Errorf("unreadable %+v, origin %v", lib.Unreadable(), lib.Origin())
	}
}

func TestRestoreStartsTheUnsavedCollectionOnce(t *testing.T) {
	lib, _, _, _ := library()
	var envs []string
	lib.Restore([]request.OpenCollection{{Environment: "dev"}, {Environment: "prod"}}, OriginRemembered, func(env string) *request.Collection {
		envs = append(envs, env)
		return newCollection("sample")
	})
	if !reflect.DeepEqual(envs, []string{"dev"}) || len(lib.Set().Entries) != 1 {
		t.Errorf("start called with %v, %d open; want once with dev", envs, len(lib.Set().Entries))
	}
}

func TestRestoreThatOpensNothingStartsWithTheRememberedEnvironmentOfTheUnsavedOne(t *testing.T) {
	lib, _, settings, _ := library()
	settings.list = []request.OpenCollection{{Folder: "/w/gone", Environment: "x"}, {Environment: "prod"}}
	var envs []string
	lib.Restore([]request.OpenCollection{{Folder: "/w/gone"}}, OriginStartFolder, func(env string) *request.Collection {
		envs = append(envs, env)
		return newCollection("sample")
	})
	if !reflect.DeepEqual(envs, []string{"prod"}) || len(lib.Set().Entries) != 1 {
		t.Errorf("start called with %v, %d open", envs, len(lib.Set().Entries))
	}
}

func TestRememberEnvironmentsWritesByOrigin(t *testing.T) {
	setup := func(origin Origin) (*Library, *fakeSettings) {
		lib, store, settings, _ := library()
		store.stored("/w/a")
		settings.list = []request.OpenCollection{{Folder: "/w/a", Environment: "prod"}, {Folder: "/w/other", Environment: "dev"}}
		lib.Restore([]request.OpenCollection{{Folder: "/w/a", Environment: "dev"}}, origin, nil)
		settings.remembers = nil
		return lib, settings
	}
	t.Run("remembered origin writes the whole list", func(t *testing.T) {
		lib, settings := setup(OriginRemembered)
		lib.RememberEnvironments()
		want := []request.OpenCollection{{Folder: "/w/a", Environment: "dev"}}
		if len(settings.remembers) != 1 || !reflect.DeepEqual(settings.list, want) {
			t.Errorf("writes %d, list %+v, want %+v", len(settings.remembers), settings.list, want)
		}
	})
	t.Run("start folder origin changes only the entries naming an open collection", func(t *testing.T) {
		lib, settings := setup(OriginStartFolder)
		lib.RememberEnvironments()
		want := []request.OpenCollection{{Folder: "/w/a", Environment: "dev"}, {Folder: "/w/other", Environment: "dev"}}
		if len(settings.remembers) != 1 || !reflect.DeepEqual(settings.list, want) {
			t.Errorf("writes %d, list %+v, want %+v", len(settings.remembers), settings.list, want)
		}
	})
	t.Run("nothing changed writes nothing", func(t *testing.T) {
		lib, settings := setup(OriginStartFolder)
		lib.RememberEnvironments()
		settings.remembers = nil
		lib.RememberEnvironments()
		if len(settings.remembers) != 0 {
			t.Errorf("%d writes, want none", len(settings.remembers))
		}
	})
}

func TestRememberedEnvMatchesAStoredRelativeFolder(t *testing.T) {
	lib, _, settings, _ := library()
	settings.list = []request.OpenCollection{{Folder: "rel/col", Environment: "prod"}}
	if got := lib.RememberedEnv(AbsDir("rel/col")); got != "prod" {
		t.Errorf("got %q, want prod", got)
	}
	if got := lib.RememberedEnv("/elsewhere"); got != "" {
		t.Errorf("got %q, want none", got)
	}
}

func TestReopenReplacesTheWholeSet(t *testing.T) {
	lib, _, settings, saves, _, _ := openTwo(t)
	loaded := request.Collection{Folder: request.Folder{Name: "c"}, Envs: []request.Environment{{Name: "dev"}}}
	show := lib.Reopen(loaded, "/w/c", "dev")
	set := lib.Set()
	if len(set.Entries) != 1 || set.Entries[0].Dir != "/w/c" || set.Entries[0].Env != 0 {
		t.Fatalf("set %+v", set.Entries)
	}
	if show == nil || show.Name != "New Request" || len(settings.remembers) != 1 || len(saves.saved) != 2 {
		t.Errorf("show %+v, writes %d, saved %d", show, len(settings.remembers), len(saves.saved))
	}
}
