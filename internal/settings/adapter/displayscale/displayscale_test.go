package displayscale

import (
	"os"
	"testing"
)

func TestParseHyprlandReadsTheFocusedMonitorsScale(t *testing.T) {
	data, err := os.ReadFile("testdata/hyprctl_monitors.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := parseMonitors(data); got != 2 {
		t.Fatalf("scale %v, want 2", got)
	}
}

func TestParseHyprlandFallsBackToTheFirstMonitorAndToUnknown(t *testing.T) {
	cases := []struct {
		name, json string
		want       float64
	}{
		{"none focused", `[{"scale":1.5,"focused":false},{"scale":2,"focused":false}]`, 1.5},
		{"focused wins", `[{"scale":1,"focused":false},{"scale":1.25,"focused":true}]`, 1.25},
		{"empty list", `[]`, 0},
		{"not json", `error`, 0},
		{"zero scale", `[{"scale":0,"focused":true}]`, 0},
	}
	for _, c := range cases {
		if got := parseMonitors([]byte(c.json)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFactorReadsTheCompositorOnlyOncePerInterval(t *testing.T) {
	now := int64(0)
	runs := 0
	system := System{
		sources: []source{func() float64 { runs++; return 2 }},
		now:     func() int64 { return now },
	}
	for range 3 {
		if got := system.Factor(); got != 2 {
			t.Fatalf("scale %v", got)
		}
	}
	if runs != 1 {
		t.Fatalf("%d queries within the interval", runs)
	}
	now += refreshNanos
	system.Factor()
	if runs != 2 {
		t.Fatalf("%d queries after the interval", runs)
	}
}

func TestParseEnvScaleReadsPositiveNumbersOnly(t *testing.T) {
	cases := []struct {
		value string
		want  float64
	}{
		{"2", 2}, {"1.5", 1.5}, {" 2 ", 2}, {"", 0}, {"0", 0}, {"-1", 0}, {"auto", 0},
	}
	for _, c := range cases {
		if got := parseEnvScale(c.value); got != c.want {
			t.Errorf("%q: got %v, want %v", c.value, got, c.want)
		}
	}
}

func TestFactorTakesTheFirstSourceThatKnowsTheScale(t *testing.T) {
	unknown := func() float64 { return 0 }
	system := System{
		sources: []source{unknown, func() float64 { return 1.5 }, func() float64 { return 3 }},
		now:     func() int64 { return 0 },
	}
	if got := system.Factor(); got != 1.5 {
		t.Fatalf("got %v, want 1.5", got)
	}
}

func TestEnvSourceReadsTheFirstVariableThatIsSet(t *testing.T) {
	env := map[string]string{"GDK_SCALE": "", "QT_SCALE_FACTOR": "1.5"}
	source := envSource(func(name string) string { return env[name] }, "GDK_SCALE", "QT_SCALE_FACTOR")
	if got := source(); got != 1.5 {
		t.Fatalf("got %v, want 1.5", got)
	}
}
