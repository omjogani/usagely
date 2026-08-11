// Command usagely shows Claude Code rate-limit usage in the desktop tray.
//
// Claude Code hands rate_limits to whatever command is configured as its
// status line. `usagely hook` sits in that slot, captures the numbers to a
// cache file, and passes stdin through to the status line you already had.
// The tray reads that cache. No network calls, no credentials.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Window is one rate-limit window as Claude Code reports it.
type Window struct {
	UsedPercent float64 `json:"used_percentage"`
	ResetsAt    int64   `json:"resets_at"` // unix seconds
}

// Snapshot is what we persist for the tray to read.
type Snapshot struct {
	FiveHour  *Window `json:"five_hour,omitempty"`
	SevenDay  *Window `json:"seven_day,omitempty"`
	UpdatedAt int64   `json:"updated_at"`
}

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "hook":
		runHook(os.Args[2:])
	case "install":
		must(runInstall())
	case "uninstall":
		must(runUninstall())
	case "", "tray":
		runTray()
	default:
		fmt.Fprintf(os.Stderr, "usagely: unknown command %q\n\nusage:\n"+
			"  usagely             run the tray indicator\n"+
			"  usagely install     add autostart + the Claude Code status line hook\n"+
			"  usagely uninstall   undo install\n"+
			"  usagely hook        capture usage from status line JSON on stdin\n", cmd)
		os.Exit(2)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "usagely:", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------- cache

func cachePath() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "usagely.json")
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "usagely.json")
}

func readSnapshot() (Snapshot, error) {
	var s Snapshot
	b, err := os.ReadFile(cachePath())
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(b, &s)
}

// writeSnapshot replaces the cache atomically, so the tray never reads a
// half-written file.
func writeSnapshot(s Snapshot) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p := cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".usagely-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// ---------------------------------------------------------------- hook

func runHook(args []string) {
	fs := flag.NewFlagSet("hook", flag.ExitOnError)
	wrap := fs.String("wrap", "", "status line command to run after capturing usage")
	_ = fs.Parse(args)

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		input = nil
	}

	// Capturing is best-effort on purpose: this runs on every status line
	// refresh, and a broken cache write must never break someone's prompt.
	capture(input, time.Now())

	if *wrap != "" {
		c := exec.Command("sh", "-c", *wrap)
		c.Stdin = bytes.NewReader(input)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				os.Exit(ee.ExitCode())
			}
			os.Exit(1)
		}
		return
	}

	// Nothing wrapped, so we are the whole status line. Print something useful.
	if s, err := readSnapshot(); err == nil {
		if line := shortLine(s, time.Now()); line != "" {
			fmt.Println(line)
		}
	}
}

// capture parses a status line payload and stores any rate limits in it.
func capture(input []byte, now time.Time) {
	var payload struct {
		RateLimits struct {
			FiveHour *Window `json:"five_hour"`
			SevenDay *Window `json:"seven_day"`
		} `json:"rate_limits"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		return
	}
	// rate_limits is absent for non-subscription sessions and before the
	// first API response. Keep the previous snapshot rather than blanking it.
	if payload.RateLimits.FiveHour == nil && payload.RateLimits.SevenDay == nil {
		return
	}
	_ = writeSnapshot(Snapshot{
		FiveHour:  payload.RateLimits.FiveHour,
		SevenDay:  payload.RateLimits.SevenDay,
		UpdatedAt: now.Unix(),
	})
}

func shortLine(s Snapshot, now time.Time) string {
	var parts []string
	if pct, _, ok := live(s.FiveHour, now); ok {
		parts = append(parts, fmt.Sprintf("5h %.0f%%", pct))
	}
	if pct, _, ok := live(s.SevenDay, now); ok {
		parts = append(parts, fmt.Sprintf("7d %.0f%%", pct))
	}
	return strings.Join(parts, " · ")
}

// ---------------------------------------------------------------- shared render helpers

// live reports a window's current state. Each window has a fixed end, so once
// resets_at has passed the correct reading is zero rather than the stale value.
func live(w *Window, now time.Time) (pct float64, resetsIn time.Duration, ok bool) {
	if w == nil {
		return 0, 0, false
	}
	until := time.Unix(w.ResetsAt, 0).Sub(now)
	if until <= 0 {
		return 0, 0, true
	}
	return w.UsedPercent, until, true
}

func humanDur(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	if h := int(d.Hours()); h >= 24 {
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
	if h := int(d.Hours()); h >= 1 {
		return fmt.Sprintf("%dh %dm", h, int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

const barCells = 10

func bar(pct float64) string {
	n := min(max(int(pct/100*barCells+0.5), 0), barCells)
	return strings.Repeat("█", n) + strings.Repeat("░", barCells-n)
}

// ---------------------------------------------------------------- install

const desktopEntry = `[Desktop Entry]
Type=Application
Name=Usagely
Comment=Claude Code usage in the tray
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`

func autostartPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "usagely.desktop"), nil
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func runInstall() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	ap, err := autostartPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ap), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(ap, fmt.Appendf(nil, desktopEntry, exe), 0o644); err != nil {
		return err
	}
	fmt.Println("autostart:   ", ap)

	changed, prev, err := patchStatusLine(exe)
	if err != nil {
		return err
	}
	switch {
	case !changed:
		fmt.Println("status line:  already installed")
	case prev == "":
		fmt.Println("status line:  installed")
	default:
		fmt.Printf("status line:  installed, wrapping %s\n", prev)
	}

	fmt.Println("\nStart it now with:  usagely &")
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME") {
		fmt.Println("\nGNOME has no system tray of its own. Install the AppIndicator extension\n" +
			"or the icon will not appear:  https://extensions.gnome.org/extension/615/appindicator-support/")
	}
	return nil
}

// patchStatusLine points Claude Code's status line at `usagely hook`, keeping
// whatever command was there as the --wrap argument.
func patchStatusLine(exe string) (changed bool, previous string, err error) {
	sp, err := settingsPath()
	if err != nil {
		return false, "", err
	}
	raw, err := os.ReadFile(sp)
	if err != nil {
		return false, "", fmt.Errorf("reading %s: %w", sp, err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return false, "", fmt.Errorf("parsing %s: %w", sp, err)
	}

	sl, _ := settings["statusLine"].(map[string]any)
	if sl == nil {
		sl = map[string]any{"type": "command"}
	}
	existing, _ := sl["command"].(string)
	if strings.Contains(existing, exe+" hook") {
		return false, "", nil
	}

	if err := os.WriteFile(sp+".usagely-backup", raw, 0o600); err != nil {
		return false, "", err
	}

	if existing == "" {
		sl["command"] = exe + " hook"
	} else {
		sl["command"] = exe + " hook --wrap " + shellQuote(existing)
	}
	sl["type"] = "command"
	settings["statusLine"] = sl

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, "", err
	}
	return true, existing, os.WriteFile(sp, append(out, '\n'), 0o600)
}

func runUninstall() error {
	if ap, err := autostartPath(); err == nil {
		if err := os.Remove(ap); err == nil {
			fmt.Println("autostart:    removed")
		}
	}

	sp, err := settingsPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(sp)
	if err != nil {
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return err
	}
	sl, _ := settings["statusLine"].(map[string]any)
	cmd, _ := sl["command"].(string)
	if !strings.Contains(cmd, " hook") || !strings.Contains(cmd, "usagely") {
		fmt.Println("status line:  not ours, left alone")
		return nil
	}

	if _, rest, found := strings.Cut(cmd, " hook --wrap "); found {
		if orig, ok := shellUnquote(rest); ok {
			sl["command"] = orig
			fmt.Printf("status line:  restored %s\n", orig)
		}
	} else {
		delete(settings, "statusLine")
		fmt.Println("status line:  removed")
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sp, append(out, '\n'), 0o600)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func shellUnquote(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return "", false
	}
	return strings.ReplaceAll(s[1:len(s)-1], `'\''`, "'"), true
}
