package appearance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// userThemes are the themes found in the user's themes folder, by id (the
// file name without .json). They are set once at startup.
var userThemes = map[string]ThemeChoice{}

// AllThemeChoices are the built-in themes followed by the user's, sorted.
func AllThemeChoices() []ThemeChoice {
	out := append([]ThemeChoice(nil), ThemeChoices...)
	ids := make([]string, 0, len(userThemes))
	for id := range userThemes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, userThemes[id])
	}
	return out
}

// BuiltinThemeNamed is the built-in theme a theme file may extend.
func BuiltinThemeNamed(id string) (widget.Theme, bool) {
	for _, c := range ThemeChoices {
		if c.ID == id {
			return c.Build(), true
		}
	}
	return widget.Theme{}, false
}

// LoadUserThemes reads every *.json file in dir as a theme laid over the
// built-in theme it extends (dark when it says nothing). A file that does
// not parse is skipped and described in problems; a missing folder is no
// problem at all.
func LoadUserThemes(dir string) (themes map[string]ThemeChoice, problems []string) {
	themes = map[string]ThemeChoice{}
	if dir == "" {
		return themes, nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return themes, nil
	}
	sort.Strings(files)
	for _, file := range files {
		id := strings.TrimSuffix(filepath.Base(file), ".json")
		data, err := os.ReadFile(file)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		baseID, err := widget.ThemeExtends(data)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		if baseID == "" {
			baseID = "dark"
		}
		base, ok := BuiltinThemeNamed(baseID)
		if !ok {
			problems = append(problems, fmt.Sprintf("theme %s extends unknown theme %q", id, baseID))
			continue
		}
		th, err := widget.ParseTheme(data, &base)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		label := th.Name
		if label == "" || label == base.Name {
			label = id
		}
		built := th
		themes[id] = ThemeChoice{ID: id, Label: label, Build: func() widget.Theme { return built }}
	}
	return themes, problems
}

// SetUserThemes sets the user's themes, as LoadUserThemes returned them.
func SetUserThemes(themes map[string]ThemeChoice) { userThemes = themes }
