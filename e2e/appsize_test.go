//go:build !release

package e2e

import (
	"os"
	"strings"
	"testing"
)

// maxAppFiles caps the non-test Go files directly in internal/app. The
// package is being dissolved (.harness/reorg.md), so it must not grow while
// the extraction tasks run. The cap only goes down.
const maxAppFiles = 22

const appDir = "../internal/app"

// countAppFiles counts the non-test .go files directly in dir, not in
// subdirectories.
func countAppFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	count := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		count++
	}
	return count
}

func TestAppPackageDoesNotGrow(t *testing.T) {
	count := countAppFiles(t, appDir)
	if count > maxAppFiles {
		t.Errorf("internal/app has %d non-test files, cap is %d (maxAppFiles in e2e/appsize_test.go). "+
			"The package is being dissolved: move code out instead of adding files. "+
			"The cap only goes down: when a task removes files, lower maxAppFiles to the new count, never raise it.",
			count, maxAppFiles)
	}
}
