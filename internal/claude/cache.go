package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CachePath is where the latest snapshot lives. The hook writes it, the tray
// reads it, and nothing else needs to know.
func CachePath() string {
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

// Read loads the last stored snapshot.
func Read() (Snapshot, error) {
	var s Snapshot
	b, err := os.ReadFile(CachePath())
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(b, &s)
}

// Write replaces the cache atomically, so the tray never reads a half-written
// file while the hook is mid-update.
func Write(s Snapshot) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	path := CachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".usagely-*")
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
	return os.Rename(tmp.Name(), path)
}
