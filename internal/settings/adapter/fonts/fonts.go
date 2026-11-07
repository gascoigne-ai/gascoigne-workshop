// Package fonts finds the font files installed for the user and groups
// them into families, so a family can be picked by name.
package fonts

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"golang.org/x/image/font/sfnt"
)

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
func Scan(dirs ...string) []fontfamily.Family {
	byName := map[string]*fontfamily.Family{}
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
				entryFor = &fontfamily.Family{Name: family}
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
	families := make([]fontfamily.Family, 0, len(byName))
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

// Load reads the regular and bold files of family. A family with no bold
// file uses its regular one for both.
func Load(family fontfamily.Family) (regular, bold []byte, err error) {
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

// System is the font store of this machine: the font folders of its OS
// under Home.
type System struct{ Home string }

// Scan implements port.FontStore.
func (system System) Scan() []fontfamily.Family {
	return Scan(SystemDirs(runtime.GOOS, system.Home)...)
}

// Load implements port.FontStore.
func (System) Load(family fontfamily.Family) (regular, bold []byte, err error) { return Load(family) }

var _ port.FontStore = System{}
