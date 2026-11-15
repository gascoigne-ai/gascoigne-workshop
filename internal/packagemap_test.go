package internal

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

var updatePackageMap = flag.Bool("update", false, "rewrite .harness/packagemap.md from the tree")

// packageMapFile is where the generated map is kept, relative to internal/.
const packageMapFile = "../.harness/packagemap.md"

// roleOrder is the order of the columns: from the inside out.
var roleOrder = []string{"model", "port", "controller", "adapter", "ui"}

// packageDirs is every directory under internal/ with a non-test Go file,
// as "domain/role/leaf".
func packageDirs(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		seen[filepath.ToSlash(filepath.Dir(path))] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

// packageMap renders the packages as a table of domains by roles, with the
// import rules of each role, all read from this package's own rules.
func packageMap(dirs []string) string {
	cells := map[string]map[string][]string{} // domain -> role -> leaf paths
	var domains []string
	var loose []string // packages that sit in no domain and role
	for _, dir := range dirs {
		parts := strings.Split(dir, "/")
		if len(parts) < 2 {
			loose = append(loose, dir)
			continue
		}
		domain, role := parts[0], parts[1]
		if _, known := canImport[role]; !known {
			loose = append(loose, dir)
			continue
		}
		if cells[domain] == nil {
			cells[domain] = map[string][]string{}
			domains = append(domains, domain)
		}
		leaf := "(root)" // the package is the role folder itself
		if len(parts) > 2 {
			leaf = strings.Join(parts[2:], "/")
		}
		cells[domain][role] = append(cells[domain][role], leaf)
	}
	sort.Strings(domains)
	var b strings.Builder
	b.WriteString("# Package map\n\nGenerated from the tree and `internal/layers_test.go` by `make packagemap`; a test fails when it is out of date. Do not edit by hand.\n\n")
	b.WriteString("Each package is `internal/<domain>/<role>/<package>`. The composition root, domain `" + compositionRoot + "`, may also import the other domains' `ui` and `controller` packages.\n\n")
	b.WriteString("| domain |")
	for _, role := range roleOrder {
		b.WriteString(" " + role + " |")
	}
	b.WriteString("\n|---|")
	for range roleOrder {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, domain := range domains {
		b.WriteString("| " + domain + " |")
		for _, role := range roleOrder {
			names := cells[domain][role]
			sort.Strings(names)
			b.WriteString(" " + strings.Join(names, ", ") + " |")
		}
		b.WriteString("\n")
	}
	b.WriteString("\nWhat a role may import, in its own domain; another domain only through its `model` or `port`:\n\n")
	for _, role := range roleOrder {
		allowed := slices.Clone(canImport[role])
		b.WriteString("- `" + role + "`: " + strings.Join(allowed, ", ") + "\n")
	}
	if len(loose) > 0 {
		b.WriteString("\nOutside the scheme: " + strings.Join(loose, ", ") + "\n")
	}
	return b.String()
}

// The map in .harness/packagemap.md is the tree as it is now; run
// `make packagemap` after adding, moving or removing a package.
func TestThePackageMapIsCurrent(t *testing.T) {
	got := packageMap(packageDirs(t))
	if *updatePackageMap {
		if err := os.WriteFile(packageMapFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(packageMapFile)
	if err != nil {
		t.Fatalf("%v: run make packagemap", err)
	}
	if string(want) != got {
		t.Fatalf("%s is out of date: run make packagemap\n\ngot:\n%s", packageMapFile, got)
	}
}

func TestThePackageMapNamesEachPackageOnce(t *testing.T) {
	dirs := []string{"request/model", "request/adapter/proto", "request/adapter/proto/ws", "workspace/ui/app", "textfmt"}
	got := packageMap(dirs)
	for _, want := range []string{"| request |", "proto, proto/ws", "| workspace |", "| (root) |", "Outside the scheme: textfmt"} {
		if !strings.Contains(got, want) {
			t.Fatalf("map lacks %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "proto/ws") != 1 {
		t.Fatalf("proto/ws is listed more than once:\n%s", got)
	}
}
