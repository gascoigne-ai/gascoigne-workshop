package systeminfo

import "testing"

var sample = Report{Sections: []Section{
	{Title: "Rendering", Rows: []Row{{"Graphics library", "Metal"}, {"Vsync", "on"}}},
	{Title: "System", Rows: []Row{{"OS", "darwin/arm64"}}},
}}

func TestTextListsEachSectionWithItsRows(t *testing.T) {
	want := "Rendering\n  Graphics library: Metal\n  Vsync: on\n\nSystem\n  OS: darwin/arm64\n"
	if got := sample.Text(); got != want {
		t.Fatalf("Text() =\n%q\nwant\n%q", got, want)
	}
}

func TestValueFindsARowByLabelInAnySection(t *testing.T) {
	if got := sample.Value("OS"); got != "darwin/arm64" {
		t.Fatalf("Value(OS) = %q", got)
	}
	if got := sample.Value("missing"); got != "" {
		t.Fatalf("Value(missing) = %q, want empty", got)
	}
}
