package respview

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestCodeLinesKeepsStructureAndColors(t *testing.T) {
	lines := CodeLines([]byte(`{"a":1}`))
	if len(lines) != 3 || !lines[0].Foldable || lines[1].Foldable {
		t.Fatalf("lines = %+v", lines)
	}
	var styles []core.Token
	for _, s := range lines[1].Spans {
		styles = append(styles, s.Style)
	}
	if styles[1] != Tokens[jsonview.Key] || styles[len(styles)-1] != Tokens[jsonview.Number] {
		t.Fatalf("styles = %v", styles)
	}
}
