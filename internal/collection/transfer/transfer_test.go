package transfer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// world is two collections:
//
//	src: x, dir (inner), sub/ (deep)   dst: x, folder/ (held)
type world struct {
	src, dst   *request.Collection
	x, inner   *request.Request
	dir        *request.Folder
	deep       *request.Request
	sub        *request.Folder // inside dir
	dstX, held *request.Request
	folder     *request.Folder
}

func newWorld() world {
	src := &request.Collection{Folder: request.Folder{Name: "src"}}
	x := src.AddRequest(request.Request{Name: "x", URL: "{{host}}/a"})
	dir := src.AddFolder("dir")
	dir.Vars = []request.KV{{Name: "dirvar", Value: "1", Enabled: true}}
	inner := dir.AddRequest(request.Request{Name: "inner", URL: "{{dirvar}}"})
	sub := dir.AddFolder("sub")
	deep := sub.AddRequest(request.Request{Name: "deep", URL: "{{host}}"})
	dst := &request.Collection{Folder: request.Folder{Name: "dst"}}
	dstX := dst.AddRequest(request.Request{Name: "x"})
	folder := dst.AddFolder("folder")
	held := folder.AddRequest(request.Request{Name: "held"})
	return world{src, dst, x, inner, dir, deep, sub, dstX, held, folder}
}

func (w world) keys() [2]string { return [2]string{tree.Key(*w.src), tree.Key(*w.dst)} }

func TestApplyMoveWithinACollectionRenamesOnClash(t *testing.T) {
	w := newWorld()
	w.dir.AddRequest(request.Request{Name: "x"})
	out := Apply(Transfer{Node: request.Of(w.x), From: w.src, To: w.src, Into: w.dir, Mode: Move})
	if out.Err != nil || !out.Renamed || out.Name != "x 2" || w.x.Name != "x 2" {
		t.Fatalf("outcome %+v, name %q", out, w.x.Name)
	}
	if !w.dir.Contains(request.Of(w.x)) || len(w.src.Requests) != 0 {
		t.Fatal("the request did not move into dir")
	}
}

func TestApplyMoveAcrossKeepsThePointerAndLeavesTheSource(t *testing.T) {
	w := newWorld()
	out := Apply(Transfer{Node: request.Of(w.x), From: w.src, To: w.dst, Into: w.folder, Mode: Move})
	if out.Err != nil || out.Renamed || out.Node.Req != w.x {
		t.Fatalf("outcome %+v", out)
	}
	if w.src.Contains(request.Of(w.x)) || !w.dst.Contains(request.Of(w.x)) {
		t.Fatal("the pointer must leave src and sit in dst")
	}
}

func TestApplyMoveAFolderCarriesChildrenAndVars(t *testing.T) {
	w := newWorld()
	out := Apply(Transfer{Node: request.Of(w.dir), From: w.src, To: w.dst, Into: &w.dst.Folder, Mode: Move})
	if out.Err != nil || out.Node.Dir != w.dir {
		t.Fatalf("outcome %+v", out)
	}
	if !w.dst.Contains(request.Of(w.deep)) || w.dir.Vars[0].Name != "dirvar" || w.src.Contains(request.Of(w.dir)) {
		t.Fatal("the folder did not carry its children and variables")
	}
}

func TestApplyCopyLeavesTheSourceAndMakesADeepNewPointer(t *testing.T) {
	w := newWorld()
	out := Apply(Transfer{Node: request.Of(w.dir), From: w.src, To: w.dst, Into: w.folder, Mode: Copy})
	if out.Err != nil || out.Node.Dir == w.dir || out.Node.Dir == nil {
		t.Fatalf("outcome %+v", out)
	}
	if !w.src.Contains(request.Of(w.dir)) || !w.folder.Contains(out.Node) {
		t.Fatal("source must stay and the copy must sit in the folder")
	}
	out.Node.Dir.Vars[0].Value = "changed"
	out.Node.Dir.Requests[0].Name = "changed"
	if w.dir.Vars[0].Value != "1" || w.inner.Name != "inner" {
		t.Fatal("the copy shares data with the original")
	}
}

func TestApplyCopyIntoTheSameFolderGivesASuffix(t *testing.T) {
	w := newWorld()
	out := Apply(Transfer{Node: request.Of(w.x), From: w.src, To: w.src, Into: &w.src.Folder, Mode: Copy})
	if out.Err != nil || !out.Renamed || out.Name != "x 2" || out.Node.Req == w.x || w.x.Name != "x" {
		t.Fatalf("outcome %+v", out)
	}
}

func TestApplyMoveIntoTheCurrentParentReorders(t *testing.T) {
	w := newWorld()
	second := w.src.AddRequest(request.Request{Name: "second"})
	out := Apply(Transfer{Node: request.Of(second), From: w.src, To: w.src, Into: &w.src.Folder, Before: request.Of(w.x), Mode: Move})
	if out.Err != nil || out.Renamed || second.Name != "second" || w.src.Requests[0] != second {
		t.Fatalf("outcome %+v order %v", out, w.src.Requests)
	}
}

func TestApplyRefusalsChangeNothing(t *testing.T) {
	stranger := &request.Request{Name: "stranger"}
	otherFolder := &request.Folder{Name: "elsewhere"}
	tests := []struct {
		name string
		make func(fixture world) Transfer
		want error
	}{
		{"the root", func(fixture world) Transfer {
			return Transfer{Node: request.Of(&fixture.src.Folder), From: fixture.src, To: fixture.dst, Into: fixture.folder}
		}, request.ErrRootMove},
		{"into itself", func(fixture world) Transfer {
			return Transfer{Node: request.Of(fixture.dir), From: fixture.src, To: fixture.src, Into: fixture.dir}
		}, request.ErrIntoItself},
		{"into its subtree", func(fixture world) Transfer {
			return Transfer{Node: request.Of(fixture.dir), From: fixture.src, To: fixture.src, Into: fixture.sub, Mode: Copy}
		}, request.ErrIntoItself},
		{"node not in From", func(fixture world) Transfer {
			return Transfer{Node: request.Of(stranger), From: fixture.src, To: fixture.dst, Into: fixture.folder}
		}, request.ErrNotInTree},
		{"into not in To", func(fixture world) Transfer {
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: otherFolder}
		}, request.ErrNotInTree},
		{"into none", func(fixture world) Transfer {
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst}
		}, request.ErrNotInTree},
		{"node none", func(fixture world) Transfer {
			return Transfer{From: fixture.src, To: fixture.dst, Into: fixture.folder}
		}, request.ErrNotInTree},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := newWorld()
			before := w.keys()
			out := Apply(test.make(w))
			if !errors.Is(out.Err, test.want) {
				t.Fatalf("Err = %v, want %v", out.Err, test.want)
			}
			if w.keys() != before {
				t.Fatal("a refusal changed a tree")
			}
		})
	}
}

func TestMissingVariables(t *testing.T) {
	envs := []request.Environment{{Name: "dev", Vars: []request.KV{{Name: "fromenv", Value: "1", Enabled: true}}}}
	tests := []struct {
		name  string
		setup func(fixture world) Transfer
		want  []string
	}{
		{"collection var found", func(fixture world) Transfer {
			fixture.dst.Vars = []request.KV{{Name: "host", Enabled: true}}
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, nil},
		{"destination folder var found", func(fixture world) Transfer {
			fixture.folder.Vars = []request.KV{{Name: "host", Enabled: true}}
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: fixture.folder}
		}, nil},
		{"active env var found", func(fixture world) Transfer {
			fixture.dst.Envs = envs
			fixture.x.URL = "{{fromenv}}"
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder, ToEnv: &fixture.dst.Envs[0]}
		}, nil},
		{"env var missing without the env", func(fixture world) Transfer {
			fixture.dst.Envs = envs
			fixture.x.URL = "{{fromenv}}"
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, []string{"fromenv"}},
		{"disabled row defines nothing", func(fixture world) Transfer {
			fixture.dst.Vars = []request.KV{{Name: "host"}}
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, []string{"host"}},
		{"secret row defines", func(fixture world) Transfer {
			fixture.dst.Vars = []request.KV{{Name: "host", Enabled: true, Secret: true}}
			return Transfer{Node: request.Of(fixture.x), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, nil},
		{"source folder var is lost by a move out", func(fixture world) Transfer {
			return Transfer{Node: request.Of(fixture.inner), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, []string{"dirvar"}},
		{"a folder carries its own vars", func(fixture world) Transfer {
			fixture.dst.Vars = []request.KV{{Name: "host", Enabled: true}}
			return Transfer{Node: request.Of(fixture.dir), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder}
		}, nil},
		{"a folder is the union in first-use order", func(fixture world) Transfer {
			fixture.inner.URL = "{{b}} {{a}}"
			fixture.deep.URL = "{{a}} {{c}}"
			return Transfer{Node: request.Of(fixture.dir), From: fixture.src, To: fixture.dst, Into: &fixture.dst.Folder, Mode: Copy}
		}, []string{"b", "a", "c"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := newWorld()
			transfer := test.setup(w)
			out := Apply(transfer)
			if out.Err != nil {
				t.Fatal(out.Err)
			}
			if len(out.Missing) == 0 && len(test.want) == 0 {
				return
			}
			if !reflect.DeepEqual(out.Missing, test.want) {
				t.Fatalf("Missing = %v, want %v", out.Missing, test.want)
			}
		})
	}
}

func TestDropToARequestInAnotherCollectionLandsBeforeItInTheTargetsTree(t *testing.T) {
	w := newWorld()
	got := Drop(w.src, w.dst, request.Of(w.x), request.Of(w.held), Move)
	want := Transfer{Node: request.Of(w.x), From: w.src, To: w.dst, Into: w.folder, Before: request.Of(w.held), Mode: Move}
	if got != want {
		t.Fatalf("Drop = %+v, want %+v", got, want)
	}
	onRoot := Drop(w.src, w.dst, request.Of(w.x), request.Of(&w.dst.Folder), Copy)
	if onRoot.Into != &w.dst.Folder || !onRoot.Before.None() || onRoot.Mode != Copy {
		t.Fatalf("drop on a collection root = %+v", onRoot)
	}
}

func TestDestinationsAreAFlatPreOrderWithDepthAndReasons(t *testing.T) {
	w := newWorld()
	set := collections.Update(collections.Update(collections.Set{}, collections.Open{Collection: w.src}), collections.Open{Collection: w.dst})
	type row struct {
		folder *request.Folder
		depth  int
		why    Why
	}
	rows := func(node request.Node, mode Mode) []row {
		var out []row
		for _, dest := range Destinations(set, node, mode) {
			out = append(out, row{dest.Folder, dest.Depth, dest.Why})
		}
		return out
	}
	tests := []struct {
		name string
		node request.Node
		mode Mode
		want []row
	}{
		{"a folder moved", request.Of(w.dir), Move, []row{
			{&w.src.Folder, 0, WhySameParent}, {w.dir, 1, WhyItself}, {w.sub, 2, WhyInside},
			{&w.dst.Folder, 0, WhyNone}, {w.folder, 1, WhyNone},
		}},
		{"a folder copied may go to its parent", request.Of(w.dir), Copy, []row{
			{&w.src.Folder, 0, WhyNone}, {w.dir, 1, WhyItself}, {w.sub, 2, WhyInside},
			{&w.dst.Folder, 0, WhyNone}, {w.folder, 1, WhyNone},
		}},
		{"a request in a subfolder", request.Of(w.deep), Move, []row{
			{&w.src.Folder, 0, WhyNone}, {w.dir, 1, WhyNone}, {w.sub, 2, WhySameParent},
			{&w.dst.Folder, 0, WhyNone}, {w.folder, 1, WhyNone},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rows(test.node, test.mode); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
	if dest := Destinations(set, request.Of(w.x), Move)[3]; dest.Collection != w.dst || dest.Folder != &w.dst.Folder {
		t.Fatalf("a collection root must name its collection: %+v", dest)
	}
}

func TestMissingTextNamesAtMostFive(t *testing.T) {
	tests := []struct {
		name    string
		missing []string
		want    string
	}{
		{"one", []string{"a"}, "Not defined in Beta: a"},
		{"five", []string{"a", "b", "c", "d", "e"}, "Not defined in Beta: a, b, c, d, e"},
		{"seven", []string{"a", "b", "c", "d", "e", "f", "g"}, "Not defined in Beta: a, b, c, d, e and 2 more"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MissingText(test.missing, "Beta"); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
