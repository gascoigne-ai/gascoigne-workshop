//go:build !release

package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const shotRunTimeout = 90 * time.Second

// The program is built once per test run; every shot starts it as a child.
var (
	buildOnce   sync.Once
	builtBinary string
	buildDir    string
	buildErr    error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if buildDir != "" {
		os.RemoveAll(buildDir)
	}
	os.Exit(code)
}

// renderShot starts the program on a virtual display with Mesa's software
// rasterizer, as `make shot` does on a desktop, and returns the PNG it
// writes. args are extra flags (-set, -menu...). Without Xvfb the test is
// skipped, except in CI where a missing backend is a failure.
func renderShot(t *testing.T, args ...string) image.Image {
	t.Helper()
	xvfb, err := exec.LookPath("xvfb-run")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("xvfb-run is required in CI: %v", err)
		}
		t.Skip("xvfb-run not installed: golden shots need a virtual display")
	}
	binary := buildProgram(t)
	sandbox := t.TempDir()
	out := filepath.Join(sandbox, "shot.png")
	flags := append([]string{"-screenshot", out, "-settings", "", "-state", "", "-themes", ""}, args...)

	ctx, cancel := context.WithTimeout(context.Background(), shotRunTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, xvfb, append([]string{"-a", "-s", fmt.Sprintf("-screen 0 %dx%dx24", shotWidth, shotHeight), binary}, flags...)...)
	command.Env = softwareEnv(sandbox)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run program: %v\n%s", err, output)
	}
	return readPNG(t, out)
}

// softwareEnv forces Mesa's llvmpipe on an X11 display and keeps the child
// away from the real user state: HOME and every XDG directory are in the
// sandbox.
func softwareEnv(sandbox string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + sandbox,
		"XDG_CONFIG_HOME=" + filepath.Join(sandbox, "config"),
		"XDG_DATA_HOME=" + filepath.Join(sandbox, "data"),
		"XDG_STATE_HOME=" + filepath.Join(sandbox, "state"),
		"XDG_CACHE_HOME=" + filepath.Join(sandbox, "cache"),
		"LIBGL_ALWAYS_SOFTWARE=1",
		"GALLIUM_DRIVER=llvmpipe",
	}
	return env
}

func buildProgram(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		buildDir, buildErr = os.MkdirTemp("", "workshop-golden-")
		if buildErr != nil {
			return
		}
		builtBinary = filepath.Join(buildDir, "workshop")
		if output, err := exec.Command("go", "build", "-o", builtBinary, ".").CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("go build: %w\n%s", err, output)
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return builtBinary
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("the program wrote no shot: %v", err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode shot: %v", err)
	}
	return img
}
