package markup

import (
	"fmt"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

type namedNumber int

func (number namedNumber) String() string { return fmt.Sprintf("no.%d", int(number)) }

// A value read with {{path}} prints as fmt.Sprint prints it, alone or between
// literals, whatever shortcut a type gets.
func TestInterpFormatsLikeFprint(t *testing.T) {
	values := []any{
		0, 7, -3, 300, int64(5), uint(9), 1.5, float32(0.25), true, false, "text", "", core.Token("muted"),
		namedNumber(4), time.Second, []int{1, 2}, nil, [2]string{"a", "b"},
	}
	for _, value := range values {
		scope := env{own: Scope{"v": value}}
		for _, source := range []string{"{{v}}", "a{{v}}", "{{v}}b", "a{{v}}b{{v}}c", " {{ v }} "} {
			parts := parseInterp(source)
			got, err := parts.eval(scope)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprint(value)
			switch source {
			case "a{{v}}":
				want = "a" + want
			case "{{v}}b":
				want += "b"
			case "a{{v}}b{{v}}c":
				want = "a" + want + "b" + want + "c"
			case " {{ v }} ":
				want = " " + want + " "
			}
			if got != want {
				t.Errorf("%q with %#v = %q, want %q", source, value, got, want)
			}
		}
	}
}
