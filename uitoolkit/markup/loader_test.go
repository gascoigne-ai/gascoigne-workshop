package markup

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const rowSource = `<row gap="4" pad="8"><text bold>{{title}}</text></row>`

func TestLoadBuildsTheSameTreeAsTheInlineTemplate(t *testing.T) {
	files := fstest.MapFS{"row.vml": {Data: []byte(rowSource)}}
	loaded, err := Load(files, "row.vml")
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{"title": "Hello"}
	if got, want := view.Dump(loaded.MustBuild(scope)), view.Dump(Must("inline", rowSource).MustBuild(scope)); got != want {
		t.Errorf("loaded template dumps\n%s\nwant\n%s", got, want)
	}
}

func TestLoadNamesTheFileAndLineOfASyntaxError(t *testing.T) {
	files := fstest.MapFS{"bad.vml": {Data: []byte("<row>\n  <nosuchelement/>\n</row>")}}
	_, err := Load(files, "bad.vml")
	if err == nil {
		t.Fatal("a template with an unknown element must not load")
	}
	if !strings.Contains(err.Error(), "bad.vml:2") {
		t.Errorf("error = %q, want it to name bad.vml and line 2", err)
	}
}

func TestLoadOfAMissingFileWrapsTheCause(t *testing.T) {
	_, err := Load(fstest.MapFS{}, "gone.vml")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want one that wraps fs.ErrNotExist", err)
	}
}

func TestMustLoadPanicsOnAMissingFile(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustLoad of a missing file must panic: an embedded file that is not there is a build mistake")
		}
	}()
	MustLoad(fstest.MapFS{}, "gone.vml")
}
