package model

import (
	"reflect"
	"testing"
)

func TestAccessesListWhatARequestDoesOnThisMachine(t *testing.T) {
	tests := []struct {
		name   string
		req    Request
		folder string
		want   []Access
	}{
		{"plain rest", Request{Protocol: "rest", URL: "https://x.test"}, "/col", nil},
		{"stdio server", Request{Protocol: "mcp", URL: "stdio: npx server --flag "}, "/col", []Access{{AccessCommand, "npx server --flag"}}},
		{"http mcp server", Request{Protocol: "mcp", URL: "https://x.test/mcp"}, "/col", nil},
		{"file body outside the collection", Request{BodyType: BodyFile, Body: "/etc/passwd"}, "/col", []Access{{AccessFile, "/etc/passwd"}}},
		{"file body inside the collection", Request{BodyType: BodyFile, Body: "/col/data/a.bin"}, "/col", nil},
		{"a sibling folder with the same prefix is outside", Request{BodyType: BodyFile, Body: "/colleague/a"}, "/col", []Access{{AccessFile, "/colleague/a"}}},
		{"dot dot leaves the folder", Request{BodyType: BodyFile, Body: "/col/../etc/x"}, "/col", []Access{{AccessFile, "/etc/x"}}},
		{"multipart files, not text, not disabled", Request{BodyType: BodyMultipart, Form: []KV{
			{Name: "a", Value: "@/home/u/pic.png", Enabled: true},
			{Name: "b", Value: "plain", Enabled: true},
			{Name: "c", Value: "@@literal", Enabled: true},
			{Name: "d", Value: "@/home/u/off", Enabled: false},
		}}, "", []Access{{AccessFile, "/home/u/pic.png"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.Accesses(tt.folder); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
