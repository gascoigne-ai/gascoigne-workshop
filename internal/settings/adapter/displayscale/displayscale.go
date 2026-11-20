// Package displayscale finds the scale a Linux desktop applies to the
// display when the window system does not report it to the window: an
// Xwayland window sees a scale of 1 on a 2x monitor. Sources are tried in
// order and the first that knows the scale wins: the compositor's own
// monitor list (Hyprland, Sway), then the toolkit variables desktops
// export (GDK_SCALE, QT_SCALE_FACTOR).
package displayscale

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	refreshInterval = 2 * time.Second // a changed desktop scale is noticed within this
	queryTimeout    = time.Second
	refreshNanos    = int64(refreshInterval)
)

// source returns a scale, or 0 when it does not know it.
type source func() float64

// System asks the running desktop for the display scale.
type System struct {
	sources []source
	now     func() int64 // nanoseconds, a fake in tests

	mu       sync.Mutex // guards the fields below
	last     int64      // when the last refresh started
	scale    float64    // the last answer, 0 until the first one lands
	asked    bool
	inFlight bool // a refresh runs; at most one at a time
}

// New returns a System for the current desktop; elsewhere than Linux it
// has no sources and never knows a scale.
func New() *System {
	system := &System{now: func() int64 { return time.Now().UnixNano() }}
	if runtime.GOOS == "linux" {
		system.sources = []source{
			monitorSource("hyprctl", "monitors", "-j"),
			monitorSource("swaymsg", "-t", "get_outputs", "-r"),
			envSource(os.Getenv, "GDK_SCALE", "QT_SCALE_FACTOR"),
		}
	}
	return system
}

// Factor is the last scale the desktop gave, 0 while unknown (before the
// first answer lands). It never runs a command itself: when the answer
// is older than the refresh interval it starts one background refresh
// (never two at once) and returns the cached value. Safe from any goroutine.
func (system *System) Factor() float64 {
	system.mu.Lock()
	defer system.mu.Unlock()
	now := system.now()
	if !system.inFlight && (!system.asked || now-system.last >= refreshNanos) {
		system.asked, system.inFlight, system.last = true, true, now
		go system.refresh()
	}
	return system.scale
}

// refresh asks the sources off the caller's goroutine: a hung compositor
// command must not stall the frame loop that calls Factor.
func (system *System) refresh() {
	var scale float64
	for _, known := range system.sources {
		if scale = known(); scale > 0 {
			break
		}
	}
	system.mu.Lock()
	system.scale, system.inFlight = scale, false
	system.mu.Unlock()
}

type monitor struct {
	Scale   float64 `json:"scale"`
	Focused bool    `json:"focused"`
}

// parseMonitors reads a compositor's monitor list as JSON (the shape
// `hyprctl monitors -j` and `swaymsg -t get_outputs -r` share): the
// focused monitor's scale, else the first one's; 0 when there is none.
func parseMonitors(data []byte) float64 {
	var monitors []monitor
	if err := json.Unmarshal(data, &monitors); err != nil || len(monitors) == 0 {
		return 0
	}
	for _, found := range monitors {
		if found.Focused {
			return max(found.Scale, 0)
		}
	}
	return max(monitors[0].Scale, 0)
}

// monitorSource runs a compositor command; a missing command (another
// desktop) is the normal case and stays silent, a failing one is logged.
func monitorSource(command string, args ...string) source {
	return func() float64 {
		path, err := exec.LookPath(command)
		if err != nil {
			return 0
		}
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, path, args...).Output()
		if err != nil {
			log.Printf("display scale: %s: %v", command, err) // the next source is tried
			return 0
		}
		return parseMonitors(out)
	}
}

// envSource reads the first of names that holds a scale above 1; 1 is the
// same as no setting.
func envSource(getenv func(string) string, names ...string) source {
	return func() float64 {
		for _, name := range names {
			if scale := parseEnvScale(getenv(name)); scale > 1 {
				return scale
			}
		}
		return 0
	}
}

// parseEnvScale reads a scale variable; 0 for empty or not a positive number.
func parseEnvScale(value string) float64 {
	scale, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || !(scale > 0) || math.IsInf(scale, 0) { // !(> 0) also drops NaN
		return 0
	}
	return scale
}
