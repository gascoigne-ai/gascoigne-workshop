package destpicker

import (
	"flag"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var update = flag.Bool("update", false, "rewrite testdata/*.dump")

// fakeHost is a window of a fixed size that records what the picker asked.
type fakeHost struct {
	picked    []transfer.Destination
	dismissed int
}

func (host *fakeHost) Size() (int, int)               { return 1000, 800 }
func (host *fakeHost) Pick(dest transfer.Destination) { host.picked = append(host.picked, dest) }
func (host *fakeHost) Dismiss()                       { host.dismissed++ }

// world is two collections: src holds the request x and the folder dir with
// the folder sub; dst holds the folder folder.
type world struct {
	src, dst    *model.Collection
	x           *model.Request
	dir, sub    *model.Folder
	folder      *model.Folder
	destination []transfer.Destination
}

func newWorld(node func(fixture world) model.Node, mode transfer.Mode) (world, *Picker) {
	src := &model.Collection{Folder: model.Folder{Name: "Source"}}
	x := src.AddRequest(model.Request{Name: "x"})
	dir := src.AddFolder("dir")
	sub := dir.AddFolder("sub")
	dst := &model.Collection{Folder: model.Folder{Name: "Target"}}
	folder := dst.AddFolder("folder")
	set := collections.Update(collections.Update(collections.Set{}, collections.Open{Collection: src}), collections.Open{Collection: dst})
	fixture := world{src: src, dst: dst, x: x, dir: dir, sub: sub, folder: folder}
	chosen := node(fixture)
	fixture.destination = transfer.Destinations(set, chosen, mode)
	stripe := func(col *model.Collection) core.Token {
		if col == src {
			return "collection.1"
		}
		return "collection.2"
	}
	return fixture, New(chosen, mode, fixture.destination, stripe)
}

func requestNode(fixture world) model.Node { return model.Of(fixture.x) }
func folderNode(fixture world) model.Node  { return model.Of(fixture.dir) }

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func layOut(t *testing.T, ctx *core.Ctx, picker *Picker, host Host) *El {
	t.Helper()
	root := Stack(picker.View(ctx, host))
	Layout(ctx, root, 1000, 800)
	return root
}

func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}

func click(t *testing.T, ctx *core.Ctx, picker *Picker, host Host, clicks int, parts ...string) {
	t.Helper()
	root := layOut(t, ctx, picker, host)
	found := findAll(root, parts...)
	if len(found) == 0 {
		t.Fatalf("nothing matches %q:\n%s", parts, Dump(root))
	}
	rect := found[0].Rect()
	ctx.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true, Clicks: clicks}
	Update(ctx, root)
}

func TestRowsComeInTreeOrderWithTheirDepthAndReasons(t *testing.T) {
	_, picker := newWorld(folderNode, transfer.Move)
	type row struct {
		name   string
		depth  int
		reason string
	}
	var got []row
	for _, each := range picker.Rows() {
		got = append(got, row{each.Name, each.Depth, each.Reason})
	}
	want := []row{
		{"Source", 0, "current folder"}, {"dir", 1, "itself"}, {"sub", 2, "inside itself"},
		{"Target", 0, ""}, {"folder", 1, ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows %v, want %v", got, want)
	}
}

func TestADisabledRowCannotBePicked(t *testing.T) {
	_, picker := newWorld(folderNode, transfer.Move)
	if picker.Select(1) || picker.Selected() != -1 {
		t.Fatalf("a disabled row was selected: %d", picker.Selected())
	}
	if !picker.Select(3) || picker.Selected() != 3 {
		t.Fatal("an enabled row was refused")
	}
}

func TestUpAndDownSkipDisabledRows(t *testing.T) {
	_, picker := newWorld(folderNode, transfer.Move)
	steps := []struct {
		delta int
		want  int
	}{{1, 3}, {1, 4}, {1, 4}, {-1, 3}, {-1, 3}}
	for _, step := range steps {
		picker.Step(step.delta)
		if picker.Selected() != step.want {
			t.Fatalf("after %+d selected %d, want %d", step.delta, picker.Selected(), step.want)
		}
	}
	_, fromBelow := newWorld(requestNode, transfer.Copy)
	fromBelow.Step(-1)
	if fromBelow.Selected() != 4 {
		t.Fatalf("Up from nothing selected %d, want the last row", fromBelow.Selected())
	}
}

func TestEnterPicksTheSelectedRowAndNothingWhenNoneIs(t *testing.T) {
	world, picker := newWorld(requestNode, transfer.Move)
	host := &fakeHost{}
	picker.Keys(core.Input{Enter: true}, host)
	if len(host.picked) != 0 {
		t.Fatal("Enter picked with nothing selected")
	}
	picker.Step(1)
	picker.Step(1)
	picker.Keys(core.Input{Enter: true}, host)
	if len(host.picked) != 1 || host.picked[0].Folder != world.sub {
		t.Fatalf("picked %+v", host.picked)
	}
}

func TestArrowKeysMoveTheSelection(t *testing.T) {
	_, picker := newWorld(requestNode, transfer.Copy)
	host := &fakeHost{}
	picker.Keys(core.Input{Edits: []core.Edit{core.EditDown}}, host)
	picker.Keys(core.Input{Edits: []core.Edit{core.EditDown}}, host)
	picker.Keys(core.Input{Edits: []core.Edit{core.EditUp}}, host)
	if picker.Selected() != 0 {
		t.Fatalf("selected %d", picker.Selected())
	}
}

func TestAClickSelectsAndADoubleClickPicks(t *testing.T) {
	world, picker := newWorld(requestNode, transfer.Move)
	host, ctx := &fakeHost{}, newCtx(t)
	click(t, ctx, picker, host, 1, `text="folder"`)
	if picker.Selected() != 4 || len(host.picked) != 0 {
		t.Fatalf("after one click selected %d, picked %d", picker.Selected(), len(host.picked))
	}
	click(t, ctx, picker, host, 2, `text="folder"`)
	if len(host.picked) != 1 || host.picked[0].Folder != world.folder {
		t.Fatalf("picked %+v", host.picked)
	}
}

func TestClickingADisabledRowDoesNothing(t *testing.T) {
	_, picker := newWorld(folderNode, transfer.Move)
	host, ctx := &fakeHost{}, newCtx(t)
	click(t, ctx, picker, host, 2, `text="sub"`)
	if picker.Selected() != -1 || len(host.picked) != 0 {
		t.Fatalf("selected %d picked %d", picker.Selected(), len(host.picked))
	}
}

func TestThePrimaryButtonWorksOnlyOnceARowIsPicked(t *testing.T) {
	world, picker := newWorld(requestNode, transfer.Copy)
	host, ctx := &fakeHost{}, newCtx(t)
	if root := layOut(t, ctx, picker, host); len(findAll(root, `text="Copy here"`)) == 0 {
		t.Fatalf("no Copy here button:\n%s", Dump(root))
	}
	click(t, ctx, picker, host, 1, `text="Copy here"`)
	if len(host.picked) != 0 {
		t.Fatal("the primary button picked with nothing selected")
	}
	picker.Select(4)
	click(t, ctx, picker, host, 1, `text="Copy here"`)
	if len(host.picked) != 1 || host.picked[0].Folder != world.folder {
		t.Fatalf("picked %+v", host.picked)
	}
}

func TestCancelAndAnOutsideClickDismiss(t *testing.T) {
	_, picker := newWorld(requestNode, transfer.Move)
	host, ctx := &fakeHost{}, newCtx(t)
	click(t, ctx, picker, host, 1, `text="Cancel"`)
	root := layOut(t, ctx, picker, host)
	ctx.In = core.Input{X: 5, Y: 5, Click: true}
	Update(ctx, root)
	if host.dismissed != 2 {
		t.Fatalf("dismissed %d times, want 2", host.dismissed)
	}
}

func TestTheTitleNamesTheModeAndTheNode(t *testing.T) {
	for _, test := range []struct {
		mode           transfer.Mode
		title, primary string
	}{{transfer.Move, "Move to", "Move here"}, {transfer.Copy, "Copy to", "Copy here"}} {
		_, picker := newWorld(requestNode, test.mode)
		dump := Dump(layOut(t, newCtx(t), picker, &fakeHost{}))
		for _, want := range []string{`"` + test.title + `"`, `"x"`, `"` + test.primary + `"`, "Variables are not changed"} {
			if !strings.Contains(dump, want) {
				t.Errorf("%v: dump lacks %s:\n%s", test.mode, want, dump)
			}
		}
	}
}

// The laid-out dialog is pinned as text (testdata/*.dump; go test -update
// rewrites them after a change that was meant).
func TestThePickerLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name    string
		node    func(world) model.Node
		mode    transfer.Mode
		select_ int
	}{
		{"move-request", requestNode, transfer.Move, -1},
		{"copy-folder-picked", folderNode, transfer.Copy, 3},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, picker := newWorld(test.node, test.mode)
			if test.select_ >= 0 {
				picker.Select(test.select_)
			}
			got := Dump(layOut(t, newCtx(t), picker, &fakeHost{}))
			file := filepath.Join("testdata", test.name+".dump")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
