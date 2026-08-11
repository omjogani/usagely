package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/omjogani/usagely/internal/claude"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=Usagely
Comment=Claude Code usage in the tray
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`

const gnomeNote = `
GNOME has no system tray of its own. If the icon does not appear, install the
AppIndicator extension (Ubuntu and Pop!_OS ship it enabled already):
https://extensions.gnome.org/extension/615/appindicator-support/`

func runInstall() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	path, err := autostartPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, fmt.Appendf(nil, desktopEntry, exe), 0o644); err != nil {
		return err
	}
	fmt.Println("autostart:    ", path)

	changed, previous, err := claude.InstallStatusLine(exe)
	if err != nil {
		return err
	}
	switch {
	case !changed:
		fmt.Println("status line:   already installed")
	case previous == "":
		fmt.Println("status line:   installed")
	default:
		fmt.Printf("status line:   installed, wrapping %s\n", previous)
	}

	fmt.Println("\nStart it now with:  usagely &")
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME") {
		fmt.Println(gnomeNote)
	}
	return nil
}

func runUninstall() error {
	path, err := autostartPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err == nil {
		fmt.Println("autostart:     removed")
	}

	restored, err := claude.RemoveStatusLine()
	if err != nil {
		return err
	}
	switch restored {
	case "":
		fmt.Println("status line:   nothing of ours to remove")
	default:
		fmt.Printf("status line:   restored %s\n", restored)
	}
	return nil
}

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
