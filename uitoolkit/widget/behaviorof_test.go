package widget

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A component option on an element of another kind used to ignore the
// option (slider) or die with a bare type assertion; it names the option
// and the element now.
func TestOptionOnWrongElementPanicsNamingTheOption(t *testing.T) {
	tests := []struct {
		name   string
		option Option
	}{
		{"FieldPad", FieldPad(3)},
		{"Placeholder", Placeholder("x")},
		{"Styled", Styled(func(string) []Span { return nil })},
		{"OnTab", OnTab(func(bool) {})},
		{"OnEnter", OnEnter(func() {})},
		{"Vertical", Vertical()},
		{"AsRatio", AsRatio()},
		{"MinFirst", MinFirst(1)},
		{"MinSecond", MinSecond(1)},
		{"Wrap", Wrap()},
		{"OnSlide", OnSlide(func(int) {})},
		{"SliderMid", SliderMid(1)},
		{"OnSlideEnd", OnSlideEnd(func(int) {})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				message := fmt.Sprint(recover())
				if !strings.Contains(message, tc.name) {
					t.Errorf("panic %q does not name the option %s", message, tc.name)
				}
			}()
			Row(Control("row"), tc.option)
		})
	}
}
