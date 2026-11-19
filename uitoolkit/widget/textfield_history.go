package widget

func (f *TextField) snapshot() snapshot {
	return snapshot{f.Text, f.Cursor, f.Anchor, f.HasSel}
}

func (f *TextField) restore(saved snapshot) {
	f.Text, f.Cursor, f.Anchor, f.HasSel = saved.text, saved.cursor, saved.anchor, saved.hasSel
	f.last = kindNone
}

// record saves the state before an edit of the given kind. Typing and
// deleting coalesce with the previous edit of the same kind.
func (f *TextField) record(kind editKind) {
	if kind != kindOther && f.last == kind {
		return
	}
	f.undo = append(f.undo, f.snapshot())
	if len(f.undo) > historyLimit {
		f.undo = f.undo[len(f.undo)-historyLimit:]
	}
	f.undo = trimToBytes(f.undo, historyBytes)
	f.redo = nil
	f.last = kind
}

// trimToBytes drops the oldest snapshots until the text they hold is at
// most limit bytes. The newest one stays whatever its size, so the last
// edit can always be undone.
func trimToBytes(history []snapshot, limit int) []snapshot {
	total := 0
	for _, entry := range history {
		total += len(entry.text)
	}
	drop := 0
	for total > limit && drop < len(history)-1 {
		total -= len(history[drop].text)
		drop++
	}
	return history[drop:]
}

// Undo restores the state before the last edit group; it reports
// whether anything changed.
func (f *TextField) Undo() bool {
	if len(f.undo) == 0 {
		return false
	}
	f.redo = append(f.redo, f.snapshot())
	f.restore(f.undo[len(f.undo)-1])
	f.undo = f.undo[:len(f.undo)-1]
	return true
}

// Redo reapplies the edit group an Undo removed.
func (f *TextField) Redo() bool {
	if len(f.redo) == 0 {
		return false
	}
	f.undo = append(f.undo, f.snapshot())
	f.restore(f.redo[len(f.redo)-1])
	f.redo = f.redo[:len(f.redo)-1]
	return true
}
