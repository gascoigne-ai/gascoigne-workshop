package widget

// The CodeView lays out and measures every line it holds, and a 2 MiB body
// is about 40k lines, so doing that on each Update and Draw cost 6-18 ms a
// call with nothing changed. Two single-entry caches keep it to once per
// change. The texts cache also holds the digit count of the widest line
// number, which Draw read on every geometry it built.
//
// Access pattern: read two or more times per frame (Update, Draw, and again
// after a reveal), written when a response arrives, a fold is toggled, or
// the width or font changes. A single entry is enough because only one
// width is live at a time; the one extra layout at a wrap switch
// (keepTopLine) replaces it once. Rejected: a per-line row index with
// prefix sums, which would make a fold toggle cheap too but is far more
// code. Revisit when a fold toggle on a view of 40k lines takes over 4 ms
// or when the view limit rises.
//
// Invalidation: SetLines bumps gen. Lines and Folded are exported, so a
// writer that bypasses SetLines is caught by what the key reads: the
// length and first element of Lines, and a fingerprint of the folded
// lines. Editing a line's spans or Num in place without SetLines is not
// caught; no caller does it.

// linesKey says which content the cached texts and layout belong to.
type linesKey struct {
	gen   uint64
	count int
	first *Line
}

func (v *CodeView) linesKey() linesKey {
	key := linesKey{gen: v.gen, count: len(v.Lines)}
	if len(v.Lines) > 0 {
		key.first = &v.Lines[0]
	}
	return key
}

// foldsKey is a fingerprint of the folded lines that does not depend on
// the map's iteration order: the sum of a scrambled index of each one.
func (v *CodeView) foldsKey() uint64 {
	var sum uint64
	for index, folded := range v.Folded {
		if folded {
			sum += (uint64(index) + 1) * 0x9E3779B97F4A7C15
		}
	}
	return sum
}

// textCache holds the runes of every line as it is written, folds ignored,
// and the digit count of the widest line number.
type textCache struct {
	key    linesKey
	runes  [][]rune
	digits int
	ok     bool
}

func (v *CodeView) runesOfLines() textCache {
	runes := make([][]rune, len(v.Lines))
	for i := range v.Lines {
		runes[i] = []rune(v.rawText(i))
	}
	return textCache{key: v.linesKey(), runes: runes, digits: v.scanNumberDigits(), ok: true}
}

// freshTexts is the texts cache, rebuilt when the lines are not the ones it
// was built from.
func (v *CodeView) freshTexts() *textCache {
	if key := v.linesKey(); !v.texts.ok || v.texts.key != key {
		v.texts = v.runesOfLines()
	}
	return &v.texts
}

// lineRunes is the text of line index as it is shown, as runes. A folded
// line shows its preview, so it is not in the cache.
func (v *CodeView) lineRunes(index int) []rune {
	if v.folded(index) {
		return []rune(v.lineText(index))
	}
	return v.freshTexts().runes[index]
}

// layoutKey is everything a layout is built from.
type layoutKey struct {
	lines linesKey
	folds uint64
	width float64
	adv   float64
}

type layoutCache struct {
	key    layoutKey
	layout wrapLayout
	ok     bool
}
