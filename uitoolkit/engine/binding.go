package engine

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// KeyNames spells keys whose ebiten name is not what users read.
var KeyNames = map[ebiten.Key]string{
	ebiten.KeyComma:     ",",
	ebiten.KeyArrowUp:   "Up",
	ebiten.KeyArrowDown: "Down",
}

// Binding is a key with modifiers; an unbound action has none.
type Binding struct {
	Key              ebiten.Key
	Ctrl, Shift, Alt bool
	Bound            bool
}

// IsModifierKey reports whether k is a modifier key itself.
func IsModifierKey(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return true
	}
	return false
}

// keyLabel is how a key is written in key strings: the friendly name
// from KeyNames, else Ebitengine's.
func keyLabel(k ebiten.Key) string {
	if name, ok := KeyNames[k]; ok {
		return name
	}
	return k.String()
}

// keyByName finds a bindable key by its lower-case name, friendly or
// Ebitengine's.
var keyByName = func() map[string]ebiten.Key {
	m := map[string]ebiten.Key{}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if IsModifierKey(k) || strings.HasPrefix(k.String(), "Key(") {
			continue
		}
		m[strings.ToLower(k.String())] = k
		m[strings.ToLower(keyLabel(k))] = k
	}
	return m
}()

// ParseBinding reads "Ctrl+Shift+Tab" (modifiers in any order, names
// case-insensitive). "" is a valid, unbound Binding.
func ParseBinding(s string) (Binding, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Binding{}, nil
	}
	parts := strings.Split(s, "+")
	var b Binding
	for _, mod := range parts[:len(parts)-1] {
		flag := map[string]*bool{"ctrl": &b.Ctrl, "control": &b.Ctrl, "alt": &b.Alt, "shift": &b.Shift}[strings.ToLower(strings.TrimSpace(mod))]
		if flag == nil || *flag {
			return Binding{}, fmt.Errorf("%q is not a modifier, or is repeated", mod)
		}
		*flag = true
	}
	key, ok := keyByName[strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))]
	if !ok {
		return Binding{}, fmt.Errorf("%q is not a key", parts[len(parts)-1])
	}
	b.Key, b.Bound = key, true
	return b, nil
}

// String writes the Binding as users read it: modifiers in the order
// Ctrl, Alt, Shift, then the key; "" when unbound.
func (b Binding) String() string {
	if !b.Bound {
		return ""
	}
	var parts []string
	if b.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if b.Alt {
		parts = append(parts, "Alt")
	}
	if b.Shift {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, keyLabel(b.Key)), "+")
}
