package proto

import (
	"strings"
	"testing"
	"time"
)

func stdioExitText(t *testing.T, mode string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, ended := stdioEnds(t, mode)
	select {
	case err := <-ended:
		return err.Error()
	case <-time.After(5 * time.Second):
		t.Fatal("the session does not end")
	}
	return ""
}

func TestMCPStdioExitShowsTheLastStderrLines(t *testing.T) {
	text := stdioExitText(t, "stderr-lines")
	if !strings.Contains(text, "line 50") {
		t.Errorf("the last stderr line is missing: %q", text)
	}
	if strings.Contains(text, "line 1\n") || strings.Contains(text, "line 30\n") {
		t.Errorf("old stderr lines are kept: %q", text)
	}
	if !strings.HasPrefix(text, "the server exited: ") || !strings.Contains(text, "\nstderr:\n") {
		t.Errorf("text = %q, want the exit error then a stderr section", text)
	}
	if n := strings.Count(text, "line "); n > mcpStderrLines {
		t.Errorf("%d stderr lines kept, want at most %d", n, mcpStderrLines)
	}
}

func TestMCPStdioExitBoundsAnOverlongStderrLine(t *testing.T) {
	text := stdioExitText(t, "stderr-huge")
	if len(text) > 2*mcpStderrLine+200 {
		t.Errorf("exit text is %d bytes, want it bounded near %d", len(text), mcpStderrLine)
	}
}

func TestTailLinesKeepsTheLastLinesAndCutsLongOnes(t *testing.T) {
	tail := newTailLines(2, 5)
	tail.Write([]byte("a\nbbbbbbbb\nc\nd"))
	if got, want := tail.String(), "bbbbb\nc\nd"; got != want {
		t.Fatalf("tail = %q, want %q", got, want)
	}
}
