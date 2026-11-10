package mcpform

import (
	"image/color"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

type fakeHost struct{ submitted int }

func (host *fakeHost) TextMenu(*TextField, func()) Option { return func(*El) {} }
func (host *fakeHost) Submit()                            { host.submitted++ }

func searchTool() model.MCPTool {
	return model.MCPTool{Name: "search", Description: "Find things", Schema: map[string]any{
		"type":     "object",
		"required": []any{"query"},
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "What to find"},
			"limit": map[string]any{"type": "integer", "default": float64(10)},
			"safe":  map[string]any{"type": "boolean"},
			"sort":  map[string]any{"type": "string", "enum": []any{"asc", "desc"}},
		},
	}}
}

func TestNewFillsDefaultsAndLeavesOptionalChoicesUnset(t *testing.T) {
	form := New(searchTool())
	if form.Inputs["limit"].Text != "10" || form.Inputs["query"].Text != "" {
		t.Errorf("inputs: limit %q query %q", form.Inputs["limit"].Text, form.Inputs["query"].Text)
	}
	if form.Choices["safe"] != "" || form.Choices["sort"] != "" {
		t.Errorf("an optional boolean and enum start unset: %v", form.Choices)
	}
}

func TestAToolWithoutPropertiesIsEmpty(t *testing.T) {
	if !New(model.MCPTool{Name: "ping"}).Empty() || New(searchTool()).Empty() {
		t.Fatal("Empty is wrong")
	}
}

func TestCommandComposesATypedCall(t *testing.T) {
	form := New(searchTool())
	form.Inputs["query"].Text = "go"
	form.Inputs["limit"].Text = "5"
	form.Choices["safe"] = "true"
	form.Choices["sort"] = "desc"
	got, err := form.Command()
	if want := `call search {"query":"go","limit":5,"safe":true,"sort":"desc"}`; err != nil || got != want {
		t.Fatalf("command %q, err %v, want %q", got, err, want)
	}
}

func TestCommandNamesTheInvalidField(t *testing.T) {
	form := New(searchTool())
	form.Inputs["query"].Text = "go"
	form.Inputs["limit"].Text = "lots"
	if _, err := form.Command(); err == nil || !strings.Contains(err.Error(), "limit must be a number") {
		t.Fatalf("err = %v", err)
	}
}

func TestViewListsEveryFieldAndTheProblem(t *testing.T) {
	form := New(searchTool())
	form.Problem = "limit must be a number"
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	root := form.View(&fakeHost{})
	Layout(ctx, root, 500, 600)
	dump := Dump(root)
	for _, want := range []string{"Find things", "query *", "limit", "safe", "sort", "limit must be a number"} {
		if !strings.Contains(dump, want) {
			t.Errorf("view lacks %q:\n%s", want, dump)
		}
	}
}

func TestArgumentsAreTheJSONObjectOfTheEntries(t *testing.T) {
	form := New(searchTool())
	form.Inputs["query"].Text = "go"
	form.Choices["safe"] = "true"
	got, err := form.Arguments()
	if want := `{"query":"go","limit":10,"safe":true}`; err != nil || got != want {
		t.Fatalf("arguments %q, err %v, want %q", got, err, want)
	}
	form.Inputs["limit"].Text = "lots"
	if _, err := form.Arguments(); err == nil {
		t.Fatal("an invalid value is refused")
	}
}
