package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeSettings points HOME at a scratch directory holding the given
// settings.json, so tests never touch the real one.
func writeSettings(t *testing.T, contents string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readCommand(t *testing.T, path string) (command string, settings map[string]any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	line, _ := settings["statusLine"].(map[string]any)
	command, _ = line["command"].(string)
	return command, settings
}

// The whole point of wrapping is that the user's own status line survives
// install and comes back on uninstall, with the rest of settings.json intact.
func TestInstallWrapsAndUninstallRestores(t *testing.T) {
	const original = "bash ~/.claude/statusline-command.sh"
	path := writeSettings(t, `{
	  "model": "opus",
	  "statusLine": {"type": "command", "command": "`+original+`"},
	  "voiceEnabled": true
	}`)
	const exe = "/home/someone/.local/bin/usagely"

	changed, previous, err := InstallStatusLine(exe)
	if err != nil {
		t.Fatalf("InstallStatusLine: %v", err)
	}
	if !changed || previous != original {
		t.Fatalf("install reported changed=%v previous=%q, want true and %q", changed, previous, original)
	}

	command, settings := readCommand(t, path)
	want := exe + " hook --wrap '" + original + "'"
	if command != want {
		t.Errorf("command = %q, want %q", command, want)
	}
	if settings["model"] != "opus" || settings["voiceEnabled"] != true {
		t.Errorf("unrelated settings were lost: %v", settings)
	}
	if _, err := os.Stat(path + ".usagely-backup"); err != nil {
		t.Errorf("no backup written: %v", err)
	}

	// Installing twice must not wrap ourselves in ourselves.
	if changed, _, err := InstallStatusLine(exe); err != nil || changed {
		t.Errorf("second install reported changed=%v (err=%v), want false", changed, err)
	}

	restored, err := RemoveStatusLine()
	if err != nil {
		t.Fatalf("RemoveStatusLine: %v", err)
	}
	if restored != original {
		t.Errorf("restored %q, want %q", restored, original)
	}
	if command, settings := readCommand(t, path); command != original || settings["model"] != "opus" {
		t.Errorf("after uninstall: command=%q settings=%v", command, settings)
	}
}

func TestInstallWithNoExistingStatusLine(t *testing.T) {
	path := writeSettings(t, `{"model": "opus"}`)
	const exe = "/home/someone/.local/bin/usagely"

	if _, previous, err := InstallStatusLine(exe); err != nil || previous != "" {
		t.Fatalf("install reported previous=%q (err=%v), want empty", previous, err)
	}
	if command, _ := readCommand(t, path); command != exe+" hook" {
		t.Errorf("command = %q, want %q", command, exe+" hook")
	}

	if restored, err := RemoveStatusLine(); err != nil || restored != "" {
		t.Fatalf("uninstall reported restored=%q (err=%v), want empty", restored, err)
	}
	if _, settings := readCommand(t, path); settings["statusLine"] != nil {
		t.Errorf("statusLine should be gone, got %v", settings["statusLine"])
	}
}

// A status line we did not install is none of our business.
func TestRemoveLeavesForeignStatusLineAlone(t *testing.T) {
	const foreign = "bash ~/.claude/statusline-command.sh"
	path := writeSettings(t, `{"statusLine": {"type": "command", "command": "`+foreign+`"}}`)

	if restored, err := RemoveStatusLine(); err != nil || restored != "" {
		t.Fatalf("restored=%q err=%v, want empty and no error", restored, err)
	}
	if command, _ := readCommand(t, path); command != foreign {
		t.Errorf("command = %q, want it untouched as %q", command, foreign)
	}
}

// install stores the previous status line inside a shell command string, so
// quoting has to survive the round trip.
func TestShellQuoteRoundTrip(t *testing.T) {
	for _, s := range []string{
		"bash ~/.claude/statusline-command.sh",
		`sh -c 'echo hi'`,
		`awk '{print $1}' | tr -d "'"`,
	} {
		got, ok := shellUnquote(shellQuote(s))
		if !ok || got != s {
			t.Errorf("round trip of %q gave %q (ok=%v)", s, got, ok)
		}
	}
}
