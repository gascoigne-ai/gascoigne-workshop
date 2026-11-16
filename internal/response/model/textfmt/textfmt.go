package textfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Duration renders d as "850ms" below a second, else "1.4s".
func Duration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", duration.Seconds())
}

// Size renders n bytes as "512B", "1.50KB" or "2.00MB".
func Size(count int) string {
	const kb = 1024
	switch {
	case count < kb:
		return fmt.Sprintf("%dB", count)
	case count < kb*kb:
		return fmt.Sprintf("%.2fKB", float64(count)/kb)
	default:
		return fmt.Sprintf("%.2fMB", float64(count)/(kb*kb))
	}
}

// CompactJSON is text as one line of JSON, with its key order and number
// text untouched; an error says where the text stops being JSON.
func CompactJSON(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("the text is empty")
	}
	// A decoder reports where the text breaks; json.Compact does not.
	decoder := json.NewDecoder(strings.NewReader(text))
	var value json.RawMessage
	err := decoder.Decode(&value)
	if err == nil && decoder.More() {
		err = fmt.Errorf("more than one JSON value, the second at position %d", decoder.InputOffset()+1)
	}
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return "", fmt.Errorf("not JSON: %s at position %d", syntax.Error(), syntax.Offset)
	}
	if err != nil {
		return "", fmt.Errorf("not JSON: %w", err)
	}
	var out bytes.Buffer
	if err := json.Compact(&out, value); err != nil {
		return "", fmt.Errorf("not JSON: %w", err)
	}
	return out.String(), nil
}
