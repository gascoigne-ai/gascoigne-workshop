package settings

import (
	"reflect"
	"testing"
)

func TestDecodeKeepsDefaultsForMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    Settings
		wantErr bool
	}{
		{"empty object gives defaults", `{}`, Default(), false},
		{"a field overrides its default", `{"layout":"stacked"}`, func() Settings { s := Default(); s.Layout = "stacked"; return s }(), false},
		{"broken json gives defaults and an error", `{"theme":`, Default(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode([]byte(tt.text))
			if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v err %v; want %+v wantErr %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestEncodeWritesOnlyWhatDiffersAndRoundTrips(t *testing.T) {
	changed := Default()
	changed.Theme = "light"
	text, err := Encode(changed)
	if err != nil {
		t.Fatal(err)
	}
	if text[len(text)-1] != '\n' {
		t.Fatalf("want a trailing newline, got %q", text)
	}
	if got, err := Decode(text); err != nil || !reflect.DeepEqual(got, changed) {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	if empty, _ := Encode(Default()); string(empty) != "{}\n" {
		t.Fatalf("defaults encode as %q, want only what differs", empty)
	}
}
