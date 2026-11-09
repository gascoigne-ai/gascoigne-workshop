// Package autosave decides when something that changes by the frame, such
// as the collection or the remembered window state, is written: once its
// content has stayed the same for a pause.
package autosave

import "time"

// Debounce tracks one piece of content by a key that changes when it does.
type Debounce struct {
	wait      time.Duration
	seen      string    // the last key seen, to notice changes
	saved     string    // the key last written
	changedAt time.Time // when the key last changed
}

// New starts with initial as both the last key seen and the one written.
func New(wait time.Duration, initial string) Debounce {
	return Debounce{wait: wait, seen: initial, saved: initial}
}

// Due is called with the current key every frame. It reports whether the
// content differs from what was written and has stayed the same for the
// pause.
func (debounce *Debounce) Due(key string, now time.Time) bool {
	if key != debounce.seen {
		debounce.seen, debounce.changedAt = key, now
		return false
	}
	return key != debounce.saved && now.Sub(debounce.changedAt) >= debounce.wait
}

// Pending reports whether key differs from what was last written.
func (debounce *Debounce) Pending(key string) bool { return key != debounce.saved }

// Saved records that key was written, or given up on.
func (debounce *Debounce) Saved(key string) { debounce.seen, debounce.saved = key, key }
