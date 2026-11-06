package packaging

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

// The bundle script cannot run here (it needs macOS tools), so the parts
// that can be checked are: the property list is well-formed, names the
// executable the script copies, and has the version placeholder the script
// replaces.
func TestMacInfoPlistIsWellFormedAndMatchesTheScript(t *testing.T) {
	data, err := os.ReadFile("macos/Info.plist")
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	for {
		if _, err := decoder.Token(); err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("Info.plist is not well-formed: %v", err)
			}
			break
		}
	}
	script, err := os.ReadFile("macos/make-app.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<string>gascoigne-workshop</string>", "<string>icon</string>", "@VERSION@"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("Info.plist lacks %s", want)
		}
	}
	for _, want := range []string{"Contents/MacOS/gascoigne-workshop", "icon.icns", "@VERSION@"} {
		if !strings.Contains(string(script), want) {
			t.Errorf("make-app.sh lacks %s", want)
		}
	}
}
