package settings

import (
	"errors"
	"testing"
)

// recordingStore is a store that counts what is written to it.
type recordingStore struct {
	writes int
	text   []byte
}

func (store *recordingStore) Load(string) (Settings, error)   { return Default(), nil }
func (store *recordingStore) Save(string, Settings) error     { store.writes++; return nil }
func (store *recordingStore) ReadText(string) ([]byte, error) { return store.text, nil }
func (store *recordingStore) Stat(string) FileStamp           { return FileStamp{} }
func (store *recordingStore) WriteText(string, []byte) error  { store.writes++; return nil }

func TestReadOnlyReadsButNeverWrites(t *testing.T) {
	inner := &recordingStore{text: []byte(`{"version": 99}`)}
	store := ReadOnly(inner)

	if text, err := store.ReadText("settings.json"); err != nil || string(text) != `{"version": 99}` {
		t.Fatalf("ReadText = %q, %v; want the file as it is", text, err)
	}
	if _, err := store.Load("settings.json"); err != nil {
		t.Fatalf("Load: %v, want reads to pass through", err)
	}
	for name, write := range map[string]func() error{
		"Save":      func() error { return store.Save("settings.json", Default()) },
		"WriteText": func() error { return store.WriteText("settings.json", []byte("{}")) },
	} {
		err := write()
		if err == nil || !errors.Is(err, errNewer) {
			t.Errorf("%s error = %v, want the newer-version refusal", name, err)
		}
	}
	if inner.writes != 0 {
		t.Errorf("%d writes reached the store, want none", inner.writes)
	}
}
