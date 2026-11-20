// Package keyrepeat finds how the desktop repeats a held key, so the
// program repeats at the same speed. Sources are tried in order and the
// first that knows both the delay and the rate wins: the compositor or the
// desktop's own setting for the running system (Hyprland, GNOME, X11,
// macOS, Windows). Where none answers the caller keeps its built-in timing.
package keyrepeat

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const queryTimeout = time.Second

// source returns the delay and the interval, or zeros when it does not know
// them.
type source func() (delay, interval time.Duration)

// System asks the running desktop for its key repeat timing.
type System struct {
	sources []source
}

// New returns a System with the sources of the current OS, as a table.
func New() System {
	return System{sources: sourcesByOS[runtime.GOOS]}
}

var sourcesByOS = map[string][]source{
	"linux":   {hyprlandSource, gnomeSource, xsetSource},
	"darwin":  {macSource},
	"windows": {windowsSource},
}

// Timing is the delay before the first repeat and the interval between
// repeats, zeros when no source knows them.
func (system System) Timing() (delay, interval time.Duration) {
	for _, known := range system.sources {
		if delay, interval = known(); delay > 0 && interval > 0 {
			return delay, interval
		}
	}
	return 0, 0
}

// run executes a command and returns its output; a missing command (another
// desktop) is silent, a failing one is logged and the next source tried.
func run(command string, args ...string) (string, bool) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		log.Printf("key repeat: %s: %v", command, err)
		return "", false
	}
	return string(out), true
}

// perSecond is the interval of a repeat rate in presses per second.
func perSecond(rate float64) time.Duration {
	if rate <= 0 {
		return 0
	}
	return time.Duration(float64(time.Second) / rate)
}

// maxSetting is the largest count (milliseconds, ticks or presses a second)
// read from a desktop: a real setting is far below it, and a larger number
// would overflow time.Duration into a negative timing.
const maxSetting = 1_000_000

func milliseconds(count int) time.Duration { return time.Duration(count) * time.Millisecond }

// --- Hyprland -------------------------------------------------------------

func hyprlandSource() (time.Duration, time.Duration) {
	rate, rateOK := run("hyprctl", "getoption", "input:repeat_rate", "-j")
	delay, delayOK := run("hyprctl", "getoption", "input:repeat_delay", "-j")
	if !rateOK || !delayOK {
		return 0, 0
	}
	return parseHyprland(delay, rate)
}

// parseHyprland reads the JSON of `hyprctl getoption`: the delay in
// milliseconds and the rate in repeats per second.
func parseHyprland(delay, rate string) (time.Duration, time.Duration) {
	delayMs, delayOK := hyprlandInt(delay)
	rateCount, rateOK := hyprlandInt(rate)
	if !delayOK || !rateOK {
		return 0, 0
	}
	return milliseconds(delayMs), perSecond(float64(rateCount))
}

func hyprlandInt(text string) (int, bool) {
	var option struct {
		Int int `json:"int"`
	}
	if err := json.Unmarshal([]byte(text), &option); err != nil || option.Int <= 0 || option.Int > maxSetting {
		return 0, false
	}
	return option.Int, true
}

// --- GNOME ----------------------------------------------------------------

func gnomeSource() (time.Duration, time.Duration) {
	const schema = "org.gnome.desktop.peripherals.keyboard"
	delay, delayOK := run("gsettings", "get", schema, "delay")
	interval, intervalOK := run("gsettings", "get", schema, "repeat-interval")
	if !delayOK || !intervalOK {
		return 0, 0
	}
	return parseGSettings(delay, interval)
}

// parseGSettings reads `gsettings get` output ("uint32 500"): the delay and
// the repeat interval, both in milliseconds.
func parseGSettings(delay, interval string) (time.Duration, time.Duration) {
	delayMs, delayOK := lastInt(delay)
	intervalMs, intervalOK := lastInt(interval)
	if !delayOK || !intervalOK {
		return 0, 0
	}
	return milliseconds(delayMs), milliseconds(intervalMs)
}

func lastInt(text string) (int, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0, false
	}
	count, err := strconv.Atoi(fields[len(fields)-1])
	return count, err == nil && count > 0 && count <= maxSetting
}

// --- X11 ------------------------------------------------------------------

var xsetRepeat = regexp.MustCompile(`auto repeat delay:\s+(\d+)\s+repeat rate:\s+(\d+)`)

func xsetSource() (time.Duration, time.Duration) {
	out, ok := run("xset", "q")
	if !ok {
		return 0, 0
	}
	return parseXset(out)
}

// parseXset reads `xset q`: "auto repeat delay: 600 repeat rate: 25", the
// delay in milliseconds and the rate per second.
func parseXset(text string) (time.Duration, time.Duration) {
	found := xsetRepeat.FindStringSubmatch(text)
	if found == nil {
		return 0, 0
	}
	delayMs, delayErr := strconv.Atoi(found[1])
	rate, rateErr := strconv.Atoi(found[2])
	if delayErr != nil || rateErr != nil || delayMs > maxSetting || rate <= 0 || rate > maxSetting {
		return 0, 0
	}
	return milliseconds(delayMs), perSecond(float64(rate))
}

// --- macOS ----------------------------------------------------------------

// macTick is the unit macOS counts key repeat settings in.
const macTick = 15 * time.Millisecond

func macSource() (time.Duration, time.Duration) {
	delay, delayOK := run("defaults", "read", "-g", "InitialKeyRepeat")
	interval, intervalOK := run("defaults", "read", "-g", "KeyRepeat")
	if !delayOK || !intervalOK {
		return 0, 0
	}
	return parseMac(delay, interval)
}

// parseMac reads `defaults read -g` output (a bare number of 15 ms ticks)
// for InitialKeyRepeat and KeyRepeat.
func parseMac(delay, interval string) (time.Duration, time.Duration) {
	delayTicks, delayOK := lastInt(delay)
	intervalTicks, intervalOK := lastInt(interval)
	if !delayOK || !intervalOK {
		return 0, 0
	}
	return time.Duration(delayTicks) * macTick, time.Duration(intervalTicks) * macTick
}

// --- Windows --------------------------------------------------------------

var regValue = regexp.MustCompile(`REG_SZ\s+(\d+)`)

func windowsSource() (time.Duration, time.Duration) {
	delay, delayOK := run("reg", "query", `HKCU\Control Panel\Keyboard`, "/v", "KeyboardDelay")
	speed, speedOK := run("reg", "query", `HKCU\Control Panel\Keyboard`, "/v", "KeyboardSpeed")
	if !delayOK || !speedOK {
		return 0, 0
	}
	return parseWindows(delay, speed)
}

// parseWindows reads `reg query` output. KeyboardDelay 0..3 is 250 ms to
// 1 s in steps of 250 ms; KeyboardSpeed 0..31 is about 2.5 to 30 repeats a
// second.
func parseWindows(delay, speed string) (time.Duration, time.Duration) {
	delayStep, delayFound := regNumber(delay)
	speedStep, speedFound := regNumber(speed)
	if !delayFound || !speedFound || delayStep > 3 || speedStep > 31 {
		return 0, 0
	}
	rate := 2.5 + float64(speedStep)*27.5/31
	return time.Duration(delayStep+1) * 250 * time.Millisecond, perSecond(rate)
}

func regNumber(text string) (int, bool) {
	found := regValue.FindStringSubmatch(text)
	if found == nil {
		return 0, false
	}
	count, err := strconv.Atoi(found[1])
	return count, err == nil
}
