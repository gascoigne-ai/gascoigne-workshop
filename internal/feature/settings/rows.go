package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"maps"
	"slices"
)

// Value is the value of one setting, typed by the kind of its row: Text for
// String, File and Enum, Flag for Bool, Number for Int, and one type for each
// of the three tables. Only this package makes Values.
type Value interface {
	// Same reports whether other is a value of the same type and content;
	// an empty table or list is the same as a missing one, as in the file.
	Same(other Value) bool
	sealed()
}

// Text is the value of a String, File or Enum setting.
type Text string

// Flag is the value of a Bool setting.
type Flag bool

// Number is the value of an Int setting.
type Number int

// ShortcutTable is the value of the Shortcuts setting: action id to keys.
type ShortcutTable map[string]string

// ColorTable is the value of the Colors setting: theme id, token, color.
type ColorTable map[string]map[string]string

// Collections is the value of the CollectionList setting.
type Collections []request.OpenCollection

func (Text) sealed()          {}
func (Flag) sealed()          {}
func (Number) sealed()        {}
func (ShortcutTable) sealed() {}
func (ColorTable) sealed()    {}
func (Collections) sealed()   {}

// Same reports whether other is the same text.
func (text Text) Same(other Value) bool { return sameComparable(text, other) }

// Same reports whether other is the same flag.
func (flag Flag) Same(other Value) bool { return sameComparable(flag, other) }

// Same reports whether other is the same number.
func (number Number) Same(other Value) bool { return sameComparable(number, other) }

// Same reports whether other has the same bindings.
func (table ShortcutTable) Same(other Value) bool {
	again, ok := other.(ShortcutTable)
	return ok && maps.Equal(table, again)
}

// Same reports whether other has the same colors.
func (table ColorTable) Same(other Value) bool {
	again, ok := other.(ColorTable)
	return ok && maps.EqualFunc(table, again, maps.Equal)
}

// Same reports whether other lists the same collections in the same order.
func (list Collections) Same(other Value) bool {
	again, ok := other.(Collections)
	return ok && slices.Equal(list, again)
}

func sameComparable[T comparable](value T, other Value) bool {
	again, ok := any(other).(T)
	return ok && value == again
}

// MarshalJSON writes each collection with environment before folder, the
// order settings.json has always had.
func (list Collections) MarshalJSON() ([]byte, error) {
	type entry struct {
		Environment string `json:"environment,omitempty"`
		Folder      string `json:"folder,omitempty"`
	}
	entries := make([]entry, len(list))
	for i, open := range list {
		entries[i] = entry{Environment: open.Environment, Folder: open.Folder}
	}
	return json.Marshal(entries)
}

// ErrUnknownKey and ErrWrongValue are what Set, With and Reset refuse with.
var (
	ErrUnknownKey = errors.New("unknown setting")
	ErrWrongValue = errors.New("wrong type of value")
)

// row is one key of settings.json: what the form shows (Field) and how the
// key is read from and written to Settings. Built only by the constructors
// below, so the Kind of the Field and the type of the accessor agree.
type row struct {
	Field
	get func(Settings) Value
	set func(*Settings, Value) error
}

// typedRow builds a row from the accessor of one Settings field: its value
// type V and the conversions to and from the field's type T.
func typedRow[V Value, T any](field Field, kind Kind, access func(*Settings) *T, toValue func(T) V, fromValue func(V) T) row {
	field.Kind = kind
	return row{
		Field: field,
		get:   func(prefs Settings) Value { return toValue(*access(&prefs)) },
		set: func(prefs *Settings, value Value) error {
			typed, ok := value.(V)
			if !ok {
				return fmt.Errorf("%w: %s wants %s, got %T", ErrWrongValue, field.Key, Expectation(field), value)
			}
			*access(prefs) = fromValue(typed)
			return nil
		},
	}
}

func textOf(text string) Text   { return Text(text) }
func stringOf(text Text) string { return string(text) }

func textRow(field Field, access func(*Settings) *string) row {
	return typedRow(field, String, access, textOf, stringOf)
}

func fileRow(field Field, access func(*Settings) *string) row {
	return typedRow(field, File, access, textOf, stringOf)
}

func enumRow(field Field, access func(*Settings) *string) row {
	return typedRow(field, Enum, access, textOf, stringOf)
}

func flagRow(field Field, access func(*Settings) *bool) row {
	return typedRow(field, Bool, access, func(on bool) Flag { return Flag(on) }, func(flag Flag) bool { return bool(flag) })
}

func intRow(field Field, access func(*Settings) *int) row {
	return typedRow(field, Int, access, func(number int) Number { return Number(number) }, func(number Number) int { return int(number) })
}

func shortcutsRow(field Field, access func(*Settings) *map[string]string) row {
	return typedRow(field, Shortcuts, access,
		func(table map[string]string) ShortcutTable { return maps.Clone(table) },
		func(table ShortcutTable) map[string]string { return maps.Clone(table) })
}

func colorsRow(field Field, access func(*Settings) *map[string]map[string]string) row {
	return typedRow(field, Colors, access,
		func(table map[string]map[string]string) ColorTable { return cloneColors(table) },
		func(table ColorTable) map[string]map[string]string { return cloneColors(table) })
}

func collectionsRow(field Field, access func(*Settings) *[]request.OpenCollection) row {
	return typedRow(field, CollectionList, access,
		func(list []request.OpenCollection) Collections { return slices.Clone(list) },
		func(list Collections) []request.OpenCollection { return slices.Clone(list) })
}

func cloneColors(table map[string]map[string]string) map[string]map[string]string {
	cloned := maps.Clone(table)
	for theme, tokens := range cloned {
		cloned[theme] = maps.Clone(tokens)
	}
	return cloned
}

// fieldsOfRows are the schema entries of rows.
func fieldsOfRows(list []row) []Field {
	fields := make([]Field, len(list))
	for i, each := range list {
		fields[i] = each.Field
	}
	return fields
}

func rowFor(key string) (row, bool) {
	for _, each := range rows {
		if each.Key == key {
			return each, true
		}
	}
	return row{}, false
}

// Get is the value of key in prefs; false for a key that is not a setting.
func Get(prefs Settings, key string) (Value, bool) {
	found, ok := rowFor(key)
	if !ok {
		return nil, false
	}
	return found.get(prefs), true
}

// Set puts value into key of prefs. A key that is not a setting or a value
// of another type than the setting holds is refused and prefs stays as it was.
func Set(prefs *Settings, key string, value Value) error {
	found, ok := rowFor(key)
	if !ok {
		return fmt.Errorf("set %s: %w", key, ErrUnknownKey)
	}
	if err := found.set(prefs, value); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

// With is prefs with key set to value; on an error prefs is returned as it was.
func With(prefs Settings, key string, value Value) (Settings, error) {
	if err := Set(&prefs, key, value); err != nil {
		return prefs, err
	}
	return prefs, nil
}

// Changed reports whether the value of key differs from the default.
func Changed(prefs Settings, key string) bool {
	found, ok := rowFor(key)
	return ok && !found.get(prefs).Same(found.get(Default()))
}

// Reset puts the default back as the value of key.
func Reset(prefs *Settings, key string) error {
	found, ok := rowFor(key)
	if !ok {
		return fmt.Errorf("reset %s: %w", key, ErrUnknownKey)
	}
	return found.set(prefs, found.get(Default()))
}

// Minimal is the settings text with only what differs from the defaults,
// grouped by section, so the file stays short and a reset leaves no trace.
func Minimal(prefs Settings) ([]byte, error) {
	defaults := Default()
	kept := map[string]Value{}
	for _, each := range rows {
		value := each.get(prefs)
		if !value.Same(each.get(defaults)) {
			kept[each.Key] = value
		}
	}
	return sectionedText(kept)
}
