package shortcutpage

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// A key binding typed into the settings file parses to a binding that
// writes and parses back to itself, or to an error. The parser lives in
// uitoolkit/engine; its target is here, where settings text reaches it.
func FuzzParseBinding(f *testing.F) {
	for _, seed := range []string{"Ctrl+Shift+Tab", "Alt+Down", "Ctrl+Space", "Ctrl+Enter", "ctrl + shift + z", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		binding, err := engine.ParseBinding(text)
		if err != nil {
			return
		}
		if again, err := engine.ParseBinding(binding.String()); err != nil || again != binding {
			t.Fatalf("%q parsed to %q, which reads back as %+v (%v)", text, binding.String(), again, err)
		}
	})
}
