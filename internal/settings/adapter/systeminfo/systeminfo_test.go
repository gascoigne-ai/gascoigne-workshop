package systeminfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestTheReportHasRenderingDisplayAndSystemFromTheGraphicsSource(t *testing.T) {
	info := Info{Version: "v1.2.3", Graphics: func() Graphics {
		return Graphics{Library: "Metal", GPUImageBytes: 3 << 20, Vsync: true, FPS: 60, TPS: 60,
			Monitor: "Built-in", MonitorW: 1512, MonitorH: 982, DeviceScale: 2, WindowW: 1000, WindowH: 700}
	}}
	report := info.Read()
	var titles []string
	for _, s := range report.Sections {
		titles = append(titles, s.Title)
	}
	if got := strings.Join(titles, ","); got != "Rendering,Display,System" {
		t.Fatalf("sections %s", got)
	}
	want := map[string]string{
		"Graphics library": "Metal", "GPU image memory": "3.0 MB", "Vsync": "on",
		"Monitor size": "1512 x 982", "Device scale": "2.00", "Window size": "1000 x 700",
		"OS": runtime.GOOS + "/" + runtime.GOARCH, "Program": "v1.2.3",
	}
	for label, value := range want {
		if got := report.Value(label); got != value {
			t.Errorf("%s = %q, want %q", label, got, value)
		}
	}
}

func TestWithoutAWindowOnlyTheSystemIsReported(t *testing.T) {
	report := Info{}.Read()
	if len(report.Sections) != 1 || report.Sections[0].Title != "System" {
		t.Fatalf("sections %+v", report.Sections)
	}
	if report.Value("Go") != runtime.Version() || report.Value("Program") != "" {
		t.Fatalf("report %+v", report)
	}
}
