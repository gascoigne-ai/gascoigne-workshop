package settings

import (
	"sort"
	"strings"
)

// Completion is one suggestion for the settings text: replace the text
// from byte From to the caret with Insert.
type Completion struct {
	Label  string // what the list shows
	Insert string
	From   int
	Detail string // what the setting is and what it expects, for the list to show
}

// container is an open object or array while scanning text.
type container struct {
	isObject   bool
	key        string // the key this container is the value of
	keys       map[string]bool
	lastKey    string
	expectKey  bool
	afterColon bool
}

// Complete suggests keys and values for the settings text before caret:
// the keys of the schema not yet used, the values an enum or a boolean
// accepts, and the action ids inside "shortcuts". actions are the ids of
// the shortcuts the program has.
func Complete(text string, caret int, actions []string) []Completion {
	caret = min(max(caret, 0), len(text))
	stack, partial, inString, stringStart := scan(text[:caret])
	if len(stack) == 0 || !stack[len(stack)-1].isObject {
		return nil
	}
	top := stack[len(stack)-1]
	isKey := top.expectKey
	if inString && !isKey && !top.afterColon {
		return nil
	}
	var candidates []string
	var quoteValues bool
	var details map[string]string
	switch {
	case isKey && len(stack) == 1:
		details = map[string]string{}
		for _, field := range Schema {
			if !top.keys[field.Key] {
				candidates = append(candidates, field.Key)
				details[field.Key] = field.Description + "\nExpects " + Expectation(field) + "."
			}
		}
	case isKey && top.key == "shortcuts":
		for _, action := range actions {
			if !top.keys[action] {
				candidates = append(candidates, action)
			}
		}
	case !isKey && len(stack) == 1 && top.afterColon:
		field, ok := FieldFor(top.lastKey)
		if !ok {
			return nil
		}
		switch field.Kind {
		case Enum:
			candidates, quoteValues = field.Allowed, true
		case Bool:
			candidates = []string{"true", "false"}
		}
		details = map[string]string{}
		for _, candidate := range candidates {
			details[candidate] = field.Title + ": " + candidate
		}
	}
	return suggestions(candidates, details, partial, inString, stringStart, caret, isKey || quoteValues)
}

// suggestions filters candidates by the typed prefix and shapes each
// insertion for where the caret is: inside a string, the closing quote is
// added; outside, the whole quoted text.
func suggestions(candidates []string, details map[string]string, partial string, inString bool, stringStart, caret int, quoted bool) []Completion {
	sort.Strings(candidates)
	var out []Completion
	for _, candidate := range candidates {
		if !strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(partial)) {
			continue
		}
		completion := Completion{Label: candidate, Insert: candidate, From: caret, Detail: details[candidate]}
		switch {
		case inString:
			completion.From = stringStart
			completion.Insert = candidate + `"`
		case quoted:
			completion.Insert = `"` + candidate + `"`
		}
		out = append(out, completion)
	}
	return out
}

// scan reads text up to the caret and reports the open containers, and,
// when the caret is inside a string, what has been typed of it and where
// its content starts.
func scan(text string) (stack []*container, partial string, inString bool, stringStart int) {
	for i := 0; i < len(text); i++ {
		char := text[i]
		if inString {
			switch {
			case char == '\\':
				i++
			case char == '"':
				inString = false
				finishString(stack, text[stringStart:i])
			}
			continue
		}
		var top *container
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		switch char {
		case '"':
			inString, stringStart = true, i+1
		case '{', '[':
			key := ""
			if top != nil {
				key = top.lastKey
				top.afterColon = false
			}
			stack = append(stack, &container{isObject: char == '{', key: key, keys: map[string]bool{}, expectKey: char == '{'})
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ':':
			if top != nil {
				top.afterColon, top.expectKey = true, false
			}
		case ',':
			if top != nil {
				top.expectKey, top.afterColon = top.isObject, false
			}
		}
	}
	if inString {
		partial = text[stringStart:]
	}
	return stack, partial, inString, stringStart
}

// finishString notes a closed string: in key position it is a key, in
// value position it ends the value.
func finishString(stack []*container, content string) {
	if len(stack) == 0 {
		return
	}
	top := stack[len(stack)-1]
	if top.isObject && top.expectKey {
		top.keys[content], top.lastKey, top.expectKey = true, content, false
		return
	}
	top.afterColon = false
}
