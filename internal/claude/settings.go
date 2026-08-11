package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SettingsPath is Claude Code's user settings file.
func SettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// InstallStatusLine points Claude Code's status line at `exe hook`, keeping
// whatever command was already there as the --wrap argument so the user's own
// status line keeps rendering.
//
// The wrapped command is stored inline in settings.json rather than in a
// config file of our own: one less piece of state, and the user can read what
// we did. settings.json is backed up alongside itself before any change.
//
// changed is false when we are already installed.
func InstallStatusLine(exe string) (changed bool, previous string, err error) {
	path, settings, raw, err := loadSettings()
	if err != nil {
		return false, "", err
	}

	line, _ := settings["statusLine"].(map[string]any)
	if line == nil {
		line = map[string]any{}
	}
	existing, _ := line["command"].(string)
	if strings.Contains(existing, exe+" hook") {
		return false, "", nil
	}

	if err := os.WriteFile(path+".usagely-backup", raw, 0o600); err != nil {
		return false, "", err
	}

	if existing == "" {
		line["command"] = exe + " hook"
	} else {
		line["command"] = exe + " hook --wrap " + shellQuote(existing)
	}
	line["type"] = "command"
	settings["statusLine"] = line

	return true, existing, saveSettings(path, settings)
}

// RemoveStatusLine undoes InstallStatusLine, putting back whatever command we
// wrapped. It leaves a status line we did not install alone.
//
// restored is the recovered command, or empty if there was nothing to put back.
func RemoveStatusLine() (restored string, err error) {
	path, settings, _, err := loadSettings()
	if err != nil {
		return "", err
	}

	line, _ := settings["statusLine"].(map[string]any)
	command, _ := line["command"].(string)
	if !strings.Contains(command, "usagely") || !strings.Contains(command, " hook") {
		return "", nil
	}

	if _, wrapped, found := strings.Cut(command, " hook --wrap "); found {
		if original, ok := shellUnquote(wrapped); ok {
			line["command"] = original
			return original, saveSettings(path, settings)
		}
	}
	delete(settings, "statusLine")
	return "", saveSettings(path, settings)
}

func loadSettings() (path string, settings map[string]any, raw []byte, err error) {
	path, err = SettingsPath()
	if err != nil {
		return "", nil, nil, err
	}
	raw, err = os.ReadFile(path)
	if err != nil {
		return "", nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	// Decoding into a map keeps every setting we do not understand intact.
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "", nil, nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return path, settings, raw, nil
}

func saveSettings(path string, settings map[string]any) error {
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
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
