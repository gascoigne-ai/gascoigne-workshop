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
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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
	last    int64
	scale   float64
	asked   bool
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

// Factor is the scale, 0 when unknown. It asks the desktop at most once
// per refresh interval and is for the UI goroutine only.
func (system *System) Factor() float64 {
	now := system.now()
	if system.asked && now-system.last < refreshNanos {
		return system.scale
	}
	system.asked, system.last, system.scale = true, now, 0
	for _, known := range system.sources {
		if system.scale = known(); system.scale > 0 {
			break
		}
	}
	return system.scale
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
	if err != nil || scale <= 0 {
		return 0
	}
	return scale
}
