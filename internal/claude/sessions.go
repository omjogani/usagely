package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const maxCacheTTL = time.Hour

type Session struct {
	ID       string
	Project  string
	Cached   int
	TTL      time.Duration
	LastSeen time.Time
	Running  bool
}

func (s Session) ExpiresAt() time.Time { return s.LastSeen.Add(s.TTL) }

func (s Session) ExpiresIn(now time.Time) time.Duration { return s.ExpiresAt().Sub(now) }

func Sessions(now time.Time) []Session {
	dir := configDir()
	if dir == "" {
		return nil
	}
	live := liveSessionIDs(dir)

	paths, _ := filepath.Glob(filepath.Join(dir, "projects", "*", "*.jsonl"))
	var out []Session
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || now.Sub(info.ModTime()) > maxCacheTTL {
			continue
		}
		session, ok := readTranscript(path, info.ModTime())
		if !ok {
			continue
		}
		session.Running = live[session.ID]
		out = append(out, session)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ExpiresAt().Before(out[j].ExpiresAt()) })
	return out
}

func readTranscript(path string, lastSeen time.Time) (Session, bool) {
	file, err := os.Open(path)
	if err != nil {
		return Session{}, false
	}
	defer file.Close()

	session := Session{
		ID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		LastSeen: lastSeen,
	}

	decoder := json.NewDecoder(file)
	for {
		var record struct {
			Cwd     string `json:"cwd"`
			Message struct {
				Usage *struct {
					CacheRead     int `json:"cache_read_input_tokens"`
					CacheCreation int `json:"cache_creation_input_tokens"`
					Buckets       struct {
						OneHour int `json:"ephemeral_1h_input_tokens"`
						FiveMin int `json:"ephemeral_5m_input_tokens"`
					} `json:"cache_creation"`
				} `json:"usage"`
			} `json:"message"`
		}
		if decoder.Decode(&record) != nil {
			break
		}
		if session.Project == "" && record.Cwd != "" {
			session.Project = filepath.Base(record.Cwd)
		}
		usage := record.Message.Usage
		if usage == nil {
			continue
		}
		session.Cached = usage.CacheRead + usage.CacheCreation
		switch {
		case usage.Buckets.OneHour > 0:
			session.TTL = time.Hour
		case usage.Buckets.FiveMin > 0:
			session.TTL = 5 * time.Minute
		}
	}

	if session.Cached == 0 || session.TTL == 0 {
		return Session{}, false
	}
	if session.Project == "" {
		session.Project = session.ID[:8]
	}
	return session, true
}

func liveSessionIDs(dir string) map[string]bool {
	live := map[string]bool{}

	paths, _ := filepath.Glob(filepath.Join(dir, "sessions", "*.json"))
	for _, path := range paths {
		var record struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
		}
		if readJSONFile(path, &record) == nil && processAlive(record.PID) {
			live[record.SessionID] = true
		}
	}

	var roster struct {
		Workers map[string]struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
		} `json:"workers"`
	}
	if readJSONFile(filepath.Join(dir, "daemon", "roster.json"), &roster) == nil {
		for _, worker := range roster.Workers {
			if processAlive(worker.PID) {
				live[worker.SessionID] = true
			}
		}
	}
	return live
}

func processAlive(pid int) bool {
	return pid > 0 && syscall.Kill(pid, 0) == nil
}

func readJSONFile(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
