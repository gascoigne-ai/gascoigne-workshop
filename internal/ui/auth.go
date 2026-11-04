package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// authChoice is one option of a chooser field.
type authChoice struct{ id, label string }

// authField is one input of an auth type: a text field, or a chooser
// when choices are given. value points at the field's place in the
// request's Auth.
type authField struct {
	id, label string
	secret    bool
	value     func(a *model.Auth) *string
	choices   []authChoice
}

// authType is a way to authenticate and the fields it needs. The table
// is the whole definition: the pane and the stored values follow it.
type authType struct {
	id, label string
	width     int
	fields    []authField
}

var authTypes = []authType{
	{id: model.AuthNone, label: "None", width: 64},
	{id: model.AuthBearer, label: "Bearer", width: 72, fields: []authField{
		{id: "token", label: "Token", secret: true, value: func(a *model.Auth) *string { return &a.Token }},
	}},
	{id: model.AuthBasic, label: "Basic", width: 64, fields: []authField{
		{id: "user", label: "User", value: func(a *model.Auth) *string { return &a.User }},
		{id: "password", label: "Password", secret: true, value: func(a *model.Auth) *string { return &a.Password }},
	}},
	{id: model.AuthAPIKey, label: "API key", width: 80, fields: []authField{
		{id: "keyname", label: "Name", value: func(a *model.Auth) *string { return &a.KeyName }},
		{id: "keyvalue", label: "Value", secret: true, value: func(a *model.Auth) *string { return &a.KeyValue }},
		{id: "keyin", label: "Add to", value: func(a *model.Auth) *string { return &a.KeyIn },
			choices: []authChoice{{model.KeyInHeader, "Header"}, {model.KeyInQuery, "Query"}}},
	}},
}

// authTypeOf is the table entry for an auth type id ("" is none).
func authTypeOf(id string) authType {
	for _, t := range authTypes {
		if t.id == id {
			return t
		}
	}
	return authTypes[0]
}

// authFieldByID finds a field in any type.
func authFieldByID(id string) (authField, bool) {
	for _, t := range authTypes {
		for _, f := range t.fields {
			if f.id == id {
				return f, true
			}
		}
	}
	return authField{}, false
}

// authInput is the text field of an Auth pane field, created on first
// use from the request's stored value. Secret fields start masked.
func (t *reqTab) authInput(id string) *w.TextField {
	if in, ok := t.auth[id]; ok {
		return in
	}
	f, _ := authFieldByID(id)
	in := new(w.TextField)
	if f.value != nil {
		*in = w.NewTextField(*f.value(&t.req.Auth))
	}
	in.Masked = f.secret && !t.showSecrets
	if t.auth == nil {
		t.auth = map[string]*w.TextField{}
	}
	t.auth[id] = in
	return in
}

// toggleSecrets shows or hides the secret fields at once.
func (t *reqTab) toggleSecrets() {
	t.showSecrets = !t.showSecrets
	for id, in := range t.auth {
		f, _ := authFieldByID(id)
		in.Masked = f.secret && !t.showSecrets
	}
}

// authPane is the Auth tab: a type chooser and the fields of that type.
func (a *App) authPane(_ pane, t *reqTab, scroll *ScrollState) Node {
	current := authTypeOf(t.req.Auth.Type)
	hasSecret := false
	for _, f := range current.fields {
		hasSecret = hasSecret || f.secret
	}
	return Scroll(scroll, Pad(w.Pad), Gap(8),
		Row(H(w.RowH), Gap(6),
			For(authTypes, func(_ int, at authType) Node {
				choose := OnClick(func() { t.req.Auth.Type = at.id })
				if at.id == current.id {
					return PrimaryButton(at.label, W(at.width), choose)
				}
				return Button(at.label, W(at.width), choose)
			}),
		),
		If(len(current.fields) == 0, Lines("No authentication", w.Muted)),
		For(current.fields, func(_ int, f authField) Node { return a.authFieldRow(t, f) }),
		If(hasSecret, Row(H(w.RowH), Gap(8), OnClick(t.toggleSecrets),
			Checkbox(t.showSecrets),
			Text("Show secrets"),
		)),
	)
}

func (a *App) authFieldRow(t *reqTab, f authField) Node {
	if f.choices != nil {
		return Row(H(w.RowH), Gap(12),
			Text(f.label, W(120)),
			Row(Gap(6), For(f.choices, func(_ int, c authChoice) Node {
				choose := OnClick(func() { *f.value(&t.req.Auth) = c.id })
				stored := *f.value(&t.req.Auth)
				if stored == c.id || (stored == "" && c.id == f.choices[0].id) {
					return PrimaryButton(c.label, W(70), choose)
				}
				return Button(c.label, W(70), choose)
			})),
		)
	}
	in := t.authInput(f.id)
	in.Masked = f.secret && !t.showSecrets
	return Row(H(w.RowH), Gap(12),
		Text(f.label, W(120)),
		Field(in, Grow(1), Bg(w.Sunken),
			OnChange(func() { *f.value(&t.req.Auth) = in.Text }),
			a.textMenuOpt(in, func() { *f.value(&t.req.Auth) = in.Text })),
	)
}
