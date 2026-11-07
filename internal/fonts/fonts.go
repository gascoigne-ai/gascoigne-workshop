// Package fonts finds the font files installed for the user and groups
// them into families, so a family can be picked by name.
package fonts

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// Family is a typeface by name with the files of its regular and bold
// styles. Bold is empty when the family has none.
type Family struct {
	Name          string
	Regular, Bold string
}

// maxFiles bounds a scan, so a huge font folder cannot stall start-up.
const maxFiles = 5000

// SystemDirs are the places a platform keeps fonts, the user's own
// included.
func SystemDirs(goos, home string) []string {
	switch goos {
	case "windows":
		return []string{`C:\Windows\Fonts`, filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "Fonts")}
	case "darwin":
		return []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library", "Fonts")}
	}
	return []string{"/usr/share/fonts", "/usr/local/share/fonts", filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts")}
}

// Scan reads the .ttf and .otf files under dirs and groups them by the
// family name inside each file, sorted by name. Folders that do not exist,
// files that cannot be read and collections (.ttc) are skipped.
func Scan(dirs ...string) []Family {
	byName := map[string]*Family{}
	seen := 0
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || seen >= maxFiles {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".ttf" && ext != ".otf" {
				return nil
			}
			seen++
			family, style, ok := identify(path)
			if !ok {
				return nil
			}
			entryFor := byName[family]
			if entryFor == nil {
				entryFor = &Family{Name: family}
				byName[family] = entryFor
			}
			switch style {
			case "Regular":
				entryFor.Regular = path
			case "Bold":
				entryFor.Bold = path
			}
			return nil
		})
	}
	families := make([]Family, 0, len(byName))
	for _, family := range byName {
		if family.Regular == "" {
			continue // italic-only or exotic styles are not offered
		}
		families = append(families, *family)
	}
	sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })
	return families
}

// identify reads the family and style names of a font file; style is
// "Regular", "Bold" or the file's own other style name.
func identify(path string) (family, style string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	font, err := sfnt.Parse(data)
	if err != nil {
		return "", "", false
	}
	var buf sfnt.Buffer
	if !hasLetters(font, &buf) {
		return "", "", false
	}
	family, err = font.Name(&buf, sfnt.NameIDFamily)
	if err != nil || family == "" {
		return "", "", false
	}
	style, err = font.Name(&buf, sfnt.NameIDSubfamily)
	if err != nil {
		style = "Regular"
	}
	return family, style, true
}

// Find is the family called name, ignoring case.
func Find(families []Family, name string) (Family, bool) {
	for _, family := range families {
		if strings.EqualFold(family.Name, strings.TrimSpace(name)) {
			return family, true
		}
	}
	return Family{}, false
}

// Matching lists up to limit families whose name contains text, ignoring
// case; empty text matches all.
func Matching(families []Family, text string, limit int) []Family {
	needle := strings.ToLower(strings.TrimSpace(text))
	var out []Family
	for _, family := range families {
		if len(out) == limit {
			break
		}
		if strings.Contains(strings.ToLower(family.Name), needle) {
			out = append(out, family)
		}
	}
	return out
}

// Load reads the regular and bold files of family. A family with no bold
// file uses its regular one for both.
func Load(family Family) (regular, bold []byte, err error) {
	if family.Regular == "" {
		return nil, nil, errors.New("the family has no regular file")
	}
	regular, err = os.ReadFile(family.Regular)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", family.Regular, err)
	}
	if family.Bold == "" {
		return regular, regular, nil
	}
	bold, err = os.ReadFile(family.Bold)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", family.Bold, err)
	}
	return regular, bold, nil
}

// hasLetters reports whether the font has a glyph for every letter of the
// basic Latin alphabet, which is what a label needs at the least. Symbol and
// icon fonts do not, and the interface drawn in them would be boxes.
func hasLetters(font *sfnt.Font, buf *sfnt.Buffer) bool {
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" {
		if index, err := font.GlyphIndex(buf, r); err != nil || index == 0 {
			return false
		}
	}
	return true
}
