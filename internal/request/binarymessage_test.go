package request

import (
	"reflect"
	"testing"
)

func TestBinaryMessage(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantData []byte
		wantOK   bool
		wantErr  bool
	}{
		{"plain text", "hello", nil, false, false},
		{"spaced bytes, mixed case", "hex: de ad BE ef", []byte{0xde, 0xad, 0xbe, 0xef}, true, false},
		{"no spaces", "hex:0102", []byte{1, 2}, true, false},
		{"empty hex", "hex:", []byte{}, true, false},
		{"not hex", "hex: xyz", nil, true, true},
		{"odd length", "hex: abc", nil, true, true},
		{"prefix must lead", " hex: 01", nil, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, ok, err := BinaryMessage(test.text)
			if ok != test.wantOK || (err != nil) != test.wantErr || !reflect.DeepEqual(data, test.wantData) {
				t.Fatalf("got %v %v %v; want %v %v err=%v", data, ok, err, test.wantData, test.wantOK, test.wantErr)
			}
		})
	}
}
