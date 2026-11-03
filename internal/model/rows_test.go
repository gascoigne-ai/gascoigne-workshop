package model

import (
	"reflect"
	"testing"
)

func kvs(names ...string) []KV {
	out := make([]KV, len(names))
	for i, n := range names {
		out[i] = KV{Name: n, Value: n + "v", Enabled: true}
	}
	return out
}

func rowNames(rows []KV) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Name)
	}
	return out
}

func TestRowOperations(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		for _, tt := range []struct {
			at   int
			want []string
		}{{0, []string{"new", "a", "b"}}, {1, []string{"a", "new", "b"}}, {2, []string{"a", "b", "new"}}, {99, []string{"a", "b", "new"}}, {-5, []string{"new", "a", "b"}}} {
			if got := rowNames(InsertRow(kvs("a", "b"), tt.at, KV{Name: "new"})); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("insert at %d: %v, want %v", tt.at, got, tt.want)
			}
		}
	})
	t.Run("duplicate keeps every field and sits right after", func(t *testing.T) {
		rows := kvs("a", "b")
		rows[0].Description = "d"
		rows[0].Enabled = false
		got := DuplicateRow(rows, 0)
		if len(got) != 3 || got[1] != got[0] || got[2].Name != "b" {
			t.Fatalf("got %+v", got)
		}
		if same := DuplicateRow(rows, 9); len(same) != 2 {
			t.Fatal("an index out of range changes nothing")
		}
	})
	t.Run("delete", func(t *testing.T) {
		if got := rowNames(DeleteRow(kvs("a", "b", "c"), 1)); !reflect.DeepEqual(got, []string{"a", "c"}) {
			t.Fatalf("got %v", got)
		}
		if got := DeleteRow(kvs("a"), 5); len(got) != 1 {
			t.Fatal("out of range changes nothing")
		}
		if got := DeleteRow(kvs("a"), -1); len(got) != 1 {
			t.Fatal("negative changes nothing")
		}
	})
	t.Run("move", func(t *testing.T) {
		for _, tt := range []struct {
			from, delta int
			want        []string
			wantIdx     int
		}{
			{1, -1, []string{"b", "a", "c"}, 0},
			{1, 1, []string{"a", "c", "b"}, 2},
			{0, -1, []string{"a", "b", "c"}, 0}, // already at the top
			{2, 1, []string{"a", "b", "c"}, 2},  // already at the bottom
			{0, 5, []string{"b", "c", "a"}, 2},  // clamped to the end
		} {
			got, idx := MoveRow(kvs("a", "b", "c"), tt.from, tt.delta)
			if !reflect.DeepEqual(rowNames(got), tt.want) || idx != tt.wantIdx {
				t.Errorf("move %d by %d: %v at %d, want %v at %d", tt.from, tt.delta, rowNames(got), idx, tt.want, tt.wantIdx)
			}
		}
	})
	t.Run("operations do not alias their input", func(t *testing.T) {
		rows := kvs("a", "b")
		_ = InsertRow(rows, 0, KV{Name: "n"})
		_ = DeleteRow(rows, 0)
		_, _ = MoveRow(rows, 0, 1)
		_ = SetAll(rows, false)
		if !reflect.DeepEqual(rowNames(rows), []string{"a", "b"}) || !rows[0].Enabled {
			t.Fatalf("the input was changed: %+v", rows)
		}
	})
}

func TestSetAllAndAllEnabled(t *testing.T) {
	rows := kvs("a", "b")
	if !AllEnabled(rows) {
		t.Fatal("both are enabled")
	}
	off := SetAll(rows, false)
	if AllEnabled(off) || off[0].Enabled || off[1].Enabled {
		t.Fatalf("off = %+v", off)
	}
	off[1].Enabled = true
	if AllEnabled(off) {
		t.Fatal("one disabled row is enough")
	}
	if AllEnabled(nil) {
		t.Fatal("no rows is not all enabled")
	}
	if !AllEnabled(SetAll(off, true)) {
		t.Fatal("SetAll(true)")
	}
}

func TestDuplicateNamesAreFlaggedCaseInsensitivelyAmongEnabledRows(t *testing.T) {
	rows := []KV{
		{Name: "X-Id", Enabled: true},
		{Name: "x-id", Enabled: true},
		{Name: "X-Id", Enabled: false},
		{Name: "", Enabled: true},
		{Name: "", Enabled: true},
		{Name: "Other", Enabled: true},
	}
	want := []bool{true, true, false, false, false, false}
	if got := DuplicateNames(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBulkRoundTrip(t *testing.T) {
	rows := []KV{
		{Name: "q", Value: "gioui", Description: "repo name", Enabled: true},
		{Name: "off", Value: "x", Enabled: false},
		{Name: "url", Value: "https://a.test:8080/p?x=1", Enabled: true},
		{Name: "pad", Value: "  two spaces", Enabled: true},
		{Name: "empty", Value: "", Enabled: true},
	}
	text := FormatBulk(rows)
	want := "q: gioui\n# off: x\nurl: https://a.test:8080/p?x=1\npad:   two spaces\nempty: "
	if text != want {
		t.Fatalf("format:\n%q\nwant\n%q", text, want)
	}
	if got := ParseBulk(text, rows); !reflect.DeepEqual(got, rows) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, rows)
	}
}

func TestParseBulk(t *testing.T) {
	old := []KV{{Name: "a", Description: "first a"}, {Name: "a", Description: "second a"}, {Name: "gone", Description: "x"}}
	tests := []struct {
		name string
		in   string
		want []KV
	}{
		{"empty text has no rows", "", nil},
		{"blank lines are skipped", "\n\n  \na: 1\n", []KV{{Name: "a", Value: "1", Description: "first a", Enabled: true}}},
		{"the name ends at the first colon", "k: a: b", []KV{{Name: "k", Value: "a: b", Enabled: true}}},
		{"no colon is a name with no value", "lonely", []KV{{Name: "lonely", Enabled: true}}},
		{"no space after the colon is fine", "k:v", []KV{{Name: "k", Value: "v", Enabled: true}}},
		{"only one space after the colon is dropped", "k:   v", []KV{{Name: "k", Value: "  v", Enabled: true}}},
		{"names are trimmed", "  k  : v", []KV{{Name: "k", Value: "v", Enabled: true}}},
		{"a hash disables", "# k: v", []KV{{Name: "k", Value: "v", Enabled: false}}},
		{"CRLF lines", "a: 1\r\nb: 2\r\n", []KV{{Name: "a", Value: "1", Description: "first a", Enabled: true}, {Name: "b", Value: "2", Enabled: true}}},
		{"descriptions follow names, each used once", "a: 1\na: 2", []KV{
			{Name: "a", Value: "1", Description: "first a", Enabled: true},
			{Name: "a", Value: "2", Description: "second a", Enabled: true},
		}},
	}
	for _, tt := range tests {
		if got := ParseBulk(tt.in, old); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, tt.want)
		}
	}
}
