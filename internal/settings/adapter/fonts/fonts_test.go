package fonts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
)

// fontDir writes the Go fonts, which are real TrueType files, under
// names that say nothing about their family.
func fontDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string][]byte{
		"a.ttf": goregular.TTF, "b.ttf": gobold.TTF,
		"sub/c.ttf": gomono.TTF, "sub/d.ttf": gomonobold.TTF,
		"readme.txt": []byte("not a font"), "broken.ttf": []byte("not a font either"),
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScanGroupsFilesIntoFamiliesByTheNamesInsideThem(t *testing.T) {
	families := Scan(fontDir(t))
	if len(families) != 2 || families[0].Name != "Go" || families[1].Name != "Go Mono" {
		t.Fatalf("families = %+v", families)
	}
	for _, family := range families {
		if family.Regular == "" || family.Bold == "" {
			t.Errorf("%s: regular %q bold %q, want both", family.Name, family.Regular, family.Bold)
		}
	}
}

func TestScanIgnoresMissingFoldersAndFilesThatAreNotFonts(t *testing.T) {
	if got := Scan(filepath.Join(t.TempDir(), "missing")); len(got) != 0 {
		t.Fatalf("families = %+v", got)
	}
}

func TestFindMatchesAFamilyNameIgnoringCase(t *testing.T) {
	families := Scan(fontDir(t))
	if family, ok := Find(families, "go mono"); !ok || family.Name != "Go Mono" {
		t.Fatalf("Find = %+v, %v", family, ok)
	}
	if _, ok := Find(families, "Comic"); ok {
		t.Fatal("found a family that is not there")
	}
}

func TestMatchingListsFamiliesContainingTheText(t *testing.T) {
	families := Scan(fontDir(t))
	got := Matching(families, "MON", 5)
	if len(got) != 1 || got[0].Name != "Go Mono" {
		t.Fatalf("Matching = %+v", got)
	}
	if got := Matching(families, "", 1); len(got) != 1 {
		t.Fatalf("limit not applied: %+v", got)
	}
}

func TestLoadReadsTheRegularAndBoldFiles(t *testing.T) {
	families := Scan(fontDir(t))
	regular, bold, err := Load(families[0])
	if err != nil || len(regular) == 0 || len(bold) == 0 {
		t.Fatalf("Load: %d %d %v", len(regular), len(bold), err)
	}
}

func TestALoneRegularFileIsUsedForBoldToo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "only.ttf"), goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	families := Scan(dir)
	regular, bold, err := Load(families[0])
	if err != nil || len(bold) != len(regular) {
		t.Fatalf("bold falls back to regular: %d %d %v", len(regular), len(bold), err)
	}
}

func TestSystemDirsNamePlacesPerPlatform(t *testing.T) {
	tests := []struct {
		goos, home string
		contains   string
	}{
		{"linux", "/home/u", "/usr/share/fonts"},
		{"linux", "/home/u", "/home/u/.local/share/fonts"},
		{"darwin", "/Users/u", "/System/Library/Fonts"},
		{"darwin", "/Users/u", "/Users/u/Library/Fonts"},
		{"windows", `C:\Users\u`, `C:\Windows\Fonts`},
	}
	for _, test := range tests {
		found := false
		for _, dir := range SystemDirs(test.goos, test.home) {
			found = found || dir == test.contains
		}
		if !found {
			t.Errorf("%s: %q missing from %v", test.goos, test.contains, SystemDirs(test.goos, test.home))
		}
	}
}

// A symbol, icon or single-script font has no glyphs for the letters a label
// needs; drawing the interface in it would show boxes, so it is not offered.
// NotoSansRunic-Regular.ttf is a real font of that kind (SIL Open Font
// License), copied from a system font folder.
func TestFamiliesWithoutTheLettersOfTheAlphabetAreNotOffered(t *testing.T) {
	dir := fontDir(t)
	data, err := os.ReadFile(filepath.Join("testdata", "NotoSansRunic-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "runic.ttf"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, family := range Scan(dir) {
		if strings.Contains(family.Name, "Runic") {
			t.Fatalf("the font without Latin letters %q is offered", family.Name)
		}
	}
	if got := Scan(dir); len(got) != 2 {
		t.Fatalf("families = %+v, want only the two text families", got)
	}
}
