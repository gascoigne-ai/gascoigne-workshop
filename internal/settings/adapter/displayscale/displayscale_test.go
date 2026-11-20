package displayscale

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

// waitFactor polls Factor until it reports want: the refresh runs in the
// background, so the answer lands a moment after the call that asked.
func waitFactor(t *testing.T, system *System, want float64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for system.Factor() != want {
		if time.Now().After(deadline) {
			t.Fatalf("scale never became %v", want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestFactorReadsTheCompositorOnlyOncePerInterval(t *testing.T) {
	var now, runs atomic.Int64
	system := System{
		sources: []source{func() float64 { runs.Add(1); return 2 }},
		now:     now.Load,
	}
	waitFactor(t, &system, 2)
	for range 3 {
		if got := system.Factor(); got != 2 {
			t.Fatalf("scale %v", got)
		}
	}
	if runs.Load() != 1 {
		t.Fatalf("%d queries within the interval", runs.Load())
	}
	now.Add(refreshNanos)
	system.Factor()
	deadline := time.Now().Add(2 * time.Second)
	for runs.Load() != 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if runs.Load() != 2 {
		t.Fatalf("%d queries after the interval", runs.Load())
	}
}

// Factor runs on the frame loop (App.Layout); a hung compositor command
// used to block it for the whole query timeout.
func TestFactorNeverRunsASourceOnTheCallerGoroutine(t *testing.T) {
	release := make(chan struct{})
	var runs atomic.Int64
	system := System{
		sources: []source{func() float64 { runs.Add(1); <-release; return 2 }},
		now:     func() int64 { return 0 },
	}
	answered := make(chan float64, 1)
	go func() { answered <- system.Factor() }()
	select {
	case got := <-answered:
		if got != 0 {
			t.Fatalf("scale %v before the first answer, want 0", got)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Factor waited for the source")
	}
	for range 5 {
		system.Factor() // one refresh in flight at a time
	}
	close(release)
	waitFactor(t, &system, 2)
	if runs.Load() != 1 {
		t.Fatalf("%d queries, want 1", runs.Load())
	}
}

func TestFactorIsSafeFromManyGoroutines(t *testing.T) {
	var now atomic.Int64
	system := System{
		sources: []source{func() float64 { return 2 }},
		now:     now.Load,
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 200 {
				now.Add(refreshNanos)
				system.Factor()
			}
		}()
	}
	group.Wait()
	waitFactor(t, &system, 2)
}

func TestParseEnvScaleReadsPositiveNumbersOnly(t *testing.T) {
	cases := []struct {
		value string
		want  float64
	}{
		{"2", 2}, {"1.5", 1.5}, {" 2 ", 2}, {"", 0}, {"0", 0}, {"-1", 0}, {"auto", 0},
		{"inf", 0}, {"+Inf", 0}, {"NaN", 0}, // an infinite scale reached the window size
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
	waitFactor(t, &system, 1.5)
}

func TestEnvSourceReadsTheFirstVariableThatIsSet(t *testing.T) {
	env := map[string]string{"GDK_SCALE": "", "QT_SCALE_FACTOR": "1.5"}
	source := envSource(func(name string) string { return env[name] }, "GDK_SCALE", "QT_SCALE_FACTOR")
	if got := source(); got != 1.5 {
		t.Fatalf("got %v, want 1.5", got)
	}
}
