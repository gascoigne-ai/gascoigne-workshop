package ws

import (
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// stampLayout is how a log line says when it happened.
const stampLayout = "15:04:05"

// Log appends event to the log of tab, one line per line of its text. An
// event without a time is stamped with now.
func (c *Component) Log(tab *tabs.Tab, event request.Event) {
	if event.Dir == request.Info && event.Kind == request.KindInitialize {
		return
	}
	mark, tok := widget.Span{Text: "•", Icon: "dot"}, w.Muted
	switch event.Dir {
	case request.Sent:
		mark, tok = widget.Span{Text: "↑", Icon: "arrow-up"}, w.Info
	case request.Received:
		mark, tok = widget.Span{Text: "↓", Icon: "arrow-down"}, w.OK
	}
	text := event.Text
	if text == "" && !event.Kind.CarriesPayload() {
		text = event.Label() // a ping has no payload
	}
	if event.Dir == request.Info && strings.HasPrefix(event.Text, "Connection lost") {
		tok = w.Error
	}
	mark.Style = w.Muted // as the stamp, so the picture is the same as when the mark was a character
	stamp := event.Time.Format(stampLayout)
	if event.Time.IsZero() {
		stamp = time.Now().Format(stampLayout)
	}
	log := &c.state(tab).log
	lines := log.Lines
	for i, part := range strings.Split(text, "\n") {
		spans := []widget.Span{{Text: stamp + " ", Style: w.Muted}, mark, {Text: " ", Style: w.Muted}}
		if i > 0 {
			spans = []widget.Span{{Text: strings.Repeat(" ", len(stamp)) + "   ", Style: w.Muted}}
		}
		spans = append(spans, widget.Span{Text: strings.TrimRight(part, "\r"), Style: textToken(event.Dir, tok)})
		lines = append(lines, widget.Line{Spans: spans})
	}
	log.SetLines(lines)
}

// textToken is the color of a log line's text: the direction's own for
// problems and notes, plain for messages.
func textToken(direction request.Direction, tok w.Token) w.Token {
	if direction == request.Info {
		return tok
	}
	return w.Text
}

// Note appends a note of the UI itself to the log of tab; an error colors
// its mark.
func (c *Component) Note(tab *tabs.Tab, text string, level severity.Level) {
	c.Log(tab, request.Event{Time: time.Now(), Dir: request.Info, Kind: request.KindInfo, Text: text})
	if level != severity.Error { // recolor the line just added
		return
	}
	lines := c.state(tab).log.Lines
	lines[len(lines)-1].Spans[1].Style = w.Error
}

// Clear empties the log of tab.
func (c *Component) Clear(tab *tabs.Tab) {
	c.state(tab).log.SetLines(nil)
}
