// Package systeminfo builds the system report from the Go runtime, the build
// information and the graphics side of Ebitengine. The graphics side is a
// function you give, so tests and tools need no display; Ebiten is the
// real one and works only while the game runs.
package systeminfo

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// Graphics is what the running window system and GPU tell.
type Graphics struct {
	Library       string // "Metal", "OpenGL", "DirectX"
	GPUImageBytes int64  // memory taken by images on the GPU
	Vsync         bool
	FPS, TPS      float64 // actual frames and ticks per second
	Monitor       string
	MonitorW      int
	MonitorH      int
	DeviceScale   float64
	WindowW       int
	WindowH       int
}

// Info makes the report. The zero value works but reports no graphics.
type Info struct {
	Graphics func() Graphics // nil when there is no window
	Version  string          // the program's version, "" when unknown
}

// Read is the report as it is now.
func (i Info) Read() settings.Report {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	system := settings.Section{Title: "System", Rows: []settings.Row{
		{Label: "OS", Value: runtime.GOOS + "/" + runtime.GOARCH},
		{Label: "CPUs", Value: fmt.Sprint(runtime.NumCPU())},
		{Label: "Go", Value: runtime.Version()},
		{Label: "Ebitengine", Value: moduleVersion("github.com/hajimehoshi/ebiten/v2")},
		{Label: "Memory in use", Value: megabytes(int64(m.Alloc))},
	}}
	if i.Version != "" {
		system.Rows = append([]settings.Row{{Label: "Program", Value: i.Version}}, system.Rows...)
	}
	if i.Graphics == nil {
		return settings.Report{Sections: []settings.Section{system}}
	}
	g := i.Graphics()
	rendering := settings.Section{Title: "Rendering", Rows: []settings.Row{
		{Label: "Graphics library", Value: orUnknown(g.Library)},
		{Label: "GPU image memory", Value: megabytes(g.GPUImageBytes)},
		{Label: "Vsync", Value: onOff(g.Vsync)},
		{Label: "Frames per second", Value: fmt.Sprintf("%.0f", g.FPS)},
		{Label: "Ticks per second", Value: fmt.Sprintf("%.0f", g.TPS)},
	}}
	display := settings.Section{Title: "Display", Rows: []settings.Row{
		{Label: "Monitor", Value: orUnknown(g.Monitor)},
		{Label: "Monitor size", Value: fmt.Sprintf("%d x %d", g.MonitorW, g.MonitorH)},
		{Label: "Device scale", Value: fmt.Sprintf("%.2f", g.DeviceScale)},
		{Label: "Window size", Value: fmt.Sprintf("%d x %d", g.WindowW, g.WindowH)},
	}}
	return settings.Report{Sections: []settings.Section{rendering, display, system}}
}

// moduleVersion is the version of a module the program was built with.
func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == path {
			return dep.Version
		}
	}
	return "unknown"
}

func megabytes(size int64) string { return fmt.Sprintf("%.1f MB", float64(size)/(1<<20)) }

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func orUnknown(text string) string {
	if text == "" {
		return "unknown"
	}
	return text
}
