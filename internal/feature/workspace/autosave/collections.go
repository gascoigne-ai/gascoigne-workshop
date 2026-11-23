package autosave

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
)

// SaveDebounce is how long a collection must stay unchanged before it is
// saved by itself.
const SaveDebounce = time.Second

// Saver writes a collection to its folder; collection.Store satisfies it.
type Saver interface {
	Save(dir string, col request.Collection) error
}

// Collections keeps when each open collection was last written and writes
// it.
type Collections struct {
	saver     Saver
	writes    *serial.Writer
	debounces map[*request.Collection]*Debounce
}

// NewCollections writes through saver, in the background on writes.
func NewCollections(saver Saver, writes *serial.Writer) *Collections {
	return &Collections{saver: saver, writes: writes, debounces: map[*request.Collection]*Debounce{}}
}

// debounce is when col was last written; a collection not seen before has
// nothing written.
func (saves *Collections) debounce(col *request.Collection) *Debounce {
	debounce := saves.debounces[col]
	if debounce == nil {
		fresh := New(SaveDebounce, "")
		debounce = &fresh
		saves.debounces[col] = debounce
	}
	return debounce
}

// Due reports whether col, whose comparable form is key, differs from what
// was written and has stayed so for SaveDebounce. Call it every frame.
func (saves *Collections) Due(col *request.Collection, key string, now time.Time) bool {
	return saves.debounce(col).Due(key, now)
}

// Pending reports whether key differs from what was last written for col.
func (saves *Collections) Pending(col *request.Collection, key string) bool {
	return saves.debounce(col).Pending(key)
}

// Saved records that col is on disk as key says, or that nothing more will
// be written for it.
func (saves *Collections) Saved(col *request.Collection, key string) {
	saves.debounce(col).Saved(key)
}

// Forget drops what is known of col, which is leaving the open set.
func (saves *Collections) Forget(col *request.Collection) { delete(saves.debounces, col) }

// Write saves a copy of col to dir off the UI goroutine. done runs on it
// with the collection's name and the outcome.
func (saves *Collections) Write(dir string, col *request.Collection, done func(name string, err error)) {
	snapshot := col.Clone() // the UI goes on editing col while the job runs
	saves.writes.Write("collection:"+dir, serial.Job{
		Run:  func() error { return saves.saver.Save(dir, snapshot) },
		Done: func(err error) { done(snapshot.Name, err) },
	})
}

// WriteNow saves col to dir before it returns and, when that worked, records
// key as written.
func (saves *Collections) WriteNow(dir string, col *request.Collection, key string) error {
	if err := saves.saver.Save(dir, *col); err != nil {
		return err
	}
	saves.Saved(col, key)
	return nil
}
