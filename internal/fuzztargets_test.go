package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fuzzFunctions lists every Fuzz function of the module as "./<dir>:<Name>".
func fuzzFunctions(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "testdata", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv == nil && strings.HasPrefix(function.Name.Name, "Fuzz") {
				found = append(found, "./"+filepath.ToSlash(relative)+":"+function.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
	return found
}

// makefileFuzzRows returns the rows of the FUZZ_TARGETS block, with the
// backslash continuations joined.
func makefileFuzzRows(makefile string) []string {
	var block []string
	inBlock := false
	for _, line := range strings.Split(makefile, "\n") {
		if strings.HasPrefix(line, "FUZZ_TARGETS") {
			inBlock = true
			line = line[strings.Index(line, "=")+1:]
		} else if !inBlock {
			continue
		}
		continued := strings.HasSuffix(line, "\\")
		block = append(block, strings.TrimSuffix(line, "\\"))
		if !continued {
			break
		}
	}
	return strings.Fields(strings.Join(block, " "))
}

// fuzzTargetProblems compares the Fuzz functions with the Makefile rows both ways.
func fuzzTargetProblems(functions, rows []string) []string {
	var problems []string
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row] {
			problems = append(problems, "FUZZ_TARGETS lists "+row+" twice")
		}
		seen[row] = true
		if !slices.Contains(functions, row) {
			problems = append(problems, "FUZZ_TARGETS row "+row+" has no Fuzz function")
		}
	}
	for _, function := range functions {
		if !seen[function] {
			problems = append(problems, "Fuzz function "+function+" is missing from FUZZ_TARGETS in the Makefile")
		}
	}
	slices.Sort(problems)
	return problems
}

func TestEveryFuzzTargetIsInTheMakefile(t *testing.T) {
	root := ".."
	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	rows := makefileFuzzRows(string(makefile))
	if len(rows) == 0 {
		t.Fatal("no FUZZ_TARGETS rows found in the Makefile")
	}
	for _, problem := range fuzzTargetProblems(fuzzFunctions(t, root), rows) {
		t.Error(problem)
	}
}

func TestFuzzTargetProblems(t *testing.T) {
	tests := []struct {
		name      string
		functions []string
		rows      []string
		want      []string
	}{
		{"in step", []string{"./a:FuzzX"}, []string{"./a:FuzzX"}, nil},
		{"function without row", []string{"./a:FuzzX", "./a:FuzzY"}, []string{"./a:FuzzX"},
			[]string{"Fuzz function ./a:FuzzY is missing from FUZZ_TARGETS in the Makefile"}},
		{"row without function", []string{"./a:FuzzX"}, []string{"./a:FuzzX", "./b:FuzzZ"},
			[]string{"FUZZ_TARGETS row ./b:FuzzZ has no Fuzz function"}},
		{"duplicate row", []string{"./a:FuzzX"}, []string{"./a:FuzzX", "./a:FuzzX"},
			[]string{"FUZZ_TARGETS lists ./a:FuzzX twice"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := fuzzTargetProblems(test.functions, test.rows)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestMakefileFuzzRowsJoinsContinuations(t *testing.T) {
	makefile := "A = 1\nFUZZ_TARGETS = \\\n\t./a:FuzzX \\\n\t./b:FuzzY\nfuzz:\n\t./c:FuzzZ\n"
	got := makefileFuzzRows(makefile)
	if want := []string{"./a:FuzzX", "./b:FuzzY"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
