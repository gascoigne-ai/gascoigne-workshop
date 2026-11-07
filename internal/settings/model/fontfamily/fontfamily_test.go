package fontfamily

import "testing"

var families = []Family{{Name: "Go", Regular: "a.ttf"}, {Name: "Go Mono", Regular: "c.ttf"}}

func TestFindMatchesAFamilyNameIgnoringCase(t *testing.T) {
	if family, ok := Find(families, "go mono"); !ok || family.Name != "Go Mono" {
		t.Fatalf("Find = %+v, %v", family, ok)
	}
	if _, ok := Find(families, "Comic"); ok {
		t.Fatal("found a family that is not there")
	}
}

func TestMatchingListsFamiliesContainingTheText(t *testing.T) {
	got := Matching(families, "MON", 5)
	if len(got) != 1 || got[0].Name != "Go Mono" {
		t.Fatalf("Matching = %+v", got)
	}
	if got := Matching(families, "", 1); len(got) != 1 {
		t.Fatalf("limit not applied: %+v", got)
	}
}
