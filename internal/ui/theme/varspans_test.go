package theme

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func TestVarSpansColorEachVariableByWhetherItResolves(t *testing.T) {
	vars := []request.KV{{Name: "known", Value: "1", Enabled: true}, {Name: "off", Value: "1"}}
	tests := []struct {
		name string
		text string
		base core.Token
		want []widget.Span
	}{
		{"url base", "{{known}}/{{missing}}/x", core.Text, []widget.Span{
			{Text: "{{known}}", Style: core.OK}, {Text: "/", Style: core.Text},
			{Text: "{{missing}}", Style: core.Error}, {Text: "/x", Style: core.Text}}},
		{"cell base is no color", "Bearer {{known}} {{missing}}", "", []widget.Span{
			{Text: "Bearer "}, {Text: "{{known}}", Style: core.OK}, {Text: " "}, {Text: "{{missing}}", Style: core.Error}}},
		{"a disabled variable is not defined", "{{off}}", core.Text, []widget.Span{{Text: "{{off}}", Style: core.Error}}},
		{"no variable is one span in the base", "plain", core.Muted, []widget.Span{{Text: "plain", Style: core.Muted}}},
		{"an unclosed brace stays literal", "a {{b", core.Text, []widget.Span{{Text: "a {{b", Style: core.Text}}},
		{"empty text has no span", "", core.Text, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := VarSpans(test.text, test.base, vars); !reflect.DeepEqual(got, test.want) {
				t.Errorf("VarSpans(%q) = %+v, want %+v", test.text, got, test.want)
			}
		})
	}
}
