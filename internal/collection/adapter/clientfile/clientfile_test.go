package clientfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const globalYAML = "servers:\n  - name: Shared\n    protocol: rest\n    url: http://global\n  - name: Both\n    protocol: rest\n    url: http://global-both\n"
const localTOML = "[[servers]]\nname = \"Both\"\nprotocol = \"rest\"\nurl = \"http://local-both\"\n"

func TestLoadMergesGlobalAndLocalWithLocalWinning(t *testing.T) {
	global, local := t.TempDir(), t.TempDir()
	write(t, global, ".workshop.yaml", globalYAML)
	write(t, local, ".workshop.toml", localTOML)

	config, err := Files{GlobalDir: global, LocalDir: local}.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	urls := map[string]string{}
	for _, server := range config.Servers {
		urls[server.Name] = server.URL
	}
	if len(urls) != 2 || urls["Shared"] != "http://global" || urls["Both"] != "http://local-both" {
		t.Fatalf("servers = %v, want Shared from global and Both from local", urls)
	}
}

func TestLoadWithoutFilesIsEmptyNotAnError(t *testing.T) {
	config, err := Files{GlobalDir: t.TempDir(), LocalDir: t.TempDir()}.Load()
	if err != nil || len(config.Servers) != 0 || len(config.Collections) != 0 {
		t.Fatalf("Load = %+v, %v; want an empty config", config, err)
	}
}

func TestLoadRejectsTwoFilesInOneDirectory(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".workshop.yaml", globalYAML)
	write(t, dir, ".workshop.json", `{"servers": []}`)

	_, err := Files{LocalDir: dir}.Load()
	if err == nil || !strings.Contains(err.Error(), ".workshop.yaml") || !strings.Contains(err.Error(), ".workshop.json") {
		t.Fatalf("Load error = %v, want it to name both files", err)
	}
}

func TestLoadNamesTheBrokenFile(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".workshop.json", "{")

	_, err := Files{GlobalDir: dir}.Load()
	if err == nil || !strings.Contains(err.Error(), ".workshop.json") {
		t.Fatalf("Load error = %v, want it to name the file", err)
	}
}

// DefaultGlobalDir must follow XDG_CONFIG_HOME so tests and users can
// redirect it; it never falls back to a real home in the tests.
func TestDefaultGlobalDirFollowsTheConfigDirectory(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("HOME", t.TempDir())
	if got, want := DefaultGlobalDir(), filepath.Join(config, "gascoigne-workshop"); got != want {
		t.Fatalf("DefaultGlobalDir = %q, want %q", got, want)
	}
}
