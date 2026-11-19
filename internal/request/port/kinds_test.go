package port

import "testing"

func TestEventKindsAreDistinct(t *testing.T) {
	kinds := []EventKind{
		KindOpen, KindClose, KindEnd, KindInitialize, KindInfo, KindError, KindText, KindBinary,
		KindPing, KindPong, KindMessage, KindRequest, KindResponse, KindNotification, KindInvalid,
	}
	seen := map[string]bool{}
	for _, kind := range kinds {
		if kind.String() == "" || seen[kind.String()] {
			t.Errorf("kind %q is empty or repeated", kind)
		}
		seen[kind.String()] = true
	}
}

func TestEventLabelIsTheServersNameOnlyForAMessage(t *testing.T) {
	cases := []struct {
		name  string
		event Event
		want  string
	}{
		{"named message", Event{Kind: KindMessage, Name: "tick"}, "tick"},
		{"unnamed message", Event{Kind: KindMessage}, "message"},
		{"a name on another kind is ignored", Event{Kind: KindPing, Name: "tick"}, "ping"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.Label(); got != tc.want {
				t.Errorf("Label = %q, want %q", got, tc.want)
			}
		})
	}
}
