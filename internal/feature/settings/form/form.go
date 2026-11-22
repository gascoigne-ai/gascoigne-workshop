// Package form is the state of the settings form and what a user's action on
// it means: a typed value is refused with a note or applied, a step stays
// inside the range of its setting, a reset drops the notes and the typed
// text. Update is pure: it returns the next Model and the Effects the view
// must carry out, so the rules run without a window.
//
// The Model keeps only the refusal notes. The typed text stays a TextField in
// the view, because it is toolkit state (caret, focus) and a controller
// imports no toolkit; the values stay in settings.Settings, so there is no
// second copy of a value to disagree with. Rejected: a Model that mirrors the
// text of every field, which would have to be synced with the TextField on
// each keystroke. Revisit when the toolkit's text state is free of drawing
// types and a controller may hold it.
package form

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// Model is what the form remembers besides the settings: why the last typed
// value of a setting was refused. The zero value is ready.
type Model struct {
	notes map[string]string
}

// Note is why the value typed for key was refused, "" when it was not.
func (model Model) Note(key string) string { return model.notes[key] }

// Msg is something the user did on the form.
type Msg interface{ formMsg() }

// Typed is a text typed into the field of a setting.
type Typed struct{ Key, Text string }

// Stepped is a click on the plus or minus button of a number setting.
type Stepped struct {
	Key   string
	Delta int
}

// Reset is a click on the reset icon of a setting.
type Reset struct{ Key string }

// Choose is a click on the button that asks for the file of a setting.
type Choose struct{ Key string }

// Picked is the file the user chose for a setting.
type Picked struct{ Key, Path string }

func (Typed) formMsg()   {}
func (Stepped) formMsg() {}
func (Reset) formMsg()   {}
func (Choose) formMsg()  {}
func (Picked) formMsg()  {}

// Effect is what the view does after an Update.
type Effect interface{ formEffect() }

// Apply sets the setting Key to Value.
type Apply struct {
	Key   string
	Value settings.Value
}

// Restore puts the default back for Key and drops it from the file.
type Restore struct{ Key string }

// Fill puts Text into the field of Key.
type Fill struct{ Key, Text string }

// ForgetTexts drops the typed text of every field.
type ForgetTexts struct{}

// ApplyFonts loads the saved fonts again.
type ApplyFonts struct{}

// AskFile asks the user for a file for Key; the answer is a Picked.
type AskFile struct{ Key, Title string }

func (Apply) formEffect()       {}
func (Restore) formEffect()     {}
func (Fill) formEffect()        {}
func (ForgetTexts) formEffect() {}
func (ApplyFonts) formEffect()  {}
func (AskFile) formEffect()     {}

// fontKeys are the settings whose change loads fonts.
var fontKeys = map[string]bool{"uiFont": true, "codeFont": true}

// Update is the next Model after msg, and what the view does about it. The
// Model it is given is not changed.
func Update(model Model, current settings.Settings, msg Msg) (Model, []Effect) {
	switch msg := msg.(type) {
	case Typed:
		return typed(model, msg.Key, msg.Text)
	case Stepped:
		return stepped(model, current, msg)
	case Reset:
		effects := []Effect{Restore{msg.Key}, ForgetTexts{}}
		if fontKeys[msg.Key] {
			effects = append(effects, ApplyFonts{})
		}
		return Model{}, effects
	case Choose:
		field, ok := settings.FieldFor(msg.Key)
		if !ok {
			return model, nil
		}
		return model, []Effect{AskFile{Key: msg.Key, Title: "Choose " + field.Title}}
	case Picked:
		fill := Fill{msg.Key, msg.Path}
		next, effects := typed(model, msg.Key, msg.Path)
		return next, append([]Effect{fill}, effects...)
	}
	return model, nil
}

// typed applies text as the value of key, or notes why it cannot be: a number
// must be whole and inside the range of its setting.
func typed(model Model, key, text string) (Model, []Effect) {
	field, ok := settings.FieldFor(key)
	if !ok {
		return model, nil
	}
	text = strings.TrimSpace(text)
	var value settings.Value = settings.Text(text)
	if field.Kind == settings.Int {
		number, err := strconv.Atoi(text)
		if err != nil || number < field.Min || number > field.Max {
			note := fmt.Sprintf("Enter a whole number between %d and %d", field.Min, field.Max)
			return model.withNote(key, note), nil
		}
		value = settings.Number(number)
	}
	return model.withNote(key, ""), []Effect{Apply{key, value}}
}

// stepped moves the number setting by the delta of step, within its range.
func stepped(model Model, current settings.Settings, step Stepped) (Model, []Effect) {
	field, ok := settings.FieldFor(step.Key)
	if !ok {
		return model, nil
	}
	value, _ := settings.Get(current, step.Key)
	number, _ := value.(settings.Number)
	next := int(number) + step.Delta
	if next < field.Min || next > field.Max {
		return model, nil
	}
	text := strconv.Itoa(next)
	model, effects := typed(model, step.Key, text)
	return model, append([]Effect{Fill{step.Key, text}}, effects...)
}

// withNote is a copy of model with the note of key set; "" clears it.
func (model Model) withNote(key, note string) Model {
	notes := make(map[string]string, len(model.notes)+1)
	for other, text := range model.notes {
		notes[other] = text
	}
	if note == "" {
		delete(notes, key)
	} else {
		notes[key] = note
	}
	return Model{notes: notes}
}

// TextOf is the text a field of key shows first: the value of the setting.
func TextOf(current settings.Settings, key string) string {
	value, _ := settings.Get(current, key)
	switch value := value.(type) {
	case settings.Text:
		return string(value)
	case settings.Number:
		return strconv.Itoa(int(value))
	}
	return ""
}
