package tray

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/omjogani/usagely/internal/claude"
)

type sent struct{ summary, body string }

func stubSend(t *testing.T) *[]sent {
	t.Helper()
	var log []sent
	original := send
	send = func(summary, body string) error {
		log = append(log, sent{summary, body})
		return nil
	}
	t.Cleanup(func() { send = original })
	return &log
}

func seedSession(t *testing.T, dir, project, id string, cached int, age time.Duration) {
	t.Helper()
	projectDir := filepath.Join(dir, "projects", project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"cwd":"/home/me/work/` + project + `"}
{"message":{"usage":{"cache_read_input_tokens":` + itoa(cached) + `,"cache_creation_input_tokens":0,"cache_creation":{"ephemeral_1h_input_tokens":1,"ephemeral_5m_input_tokens":0}}}}
`
	path := filepath.Join(projectDir, id+".jsonl")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestWarnerFiresOncePerExpiry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	seedSession(t, dir, "Widget", "aaaa1111-0000-0000-0000-000000000000", 150000, 53*time.Minute)

	log := stubSend(t)
	w := newWarner()
	w.check(time.Now())
	w.check(time.Now())

	if len(*log) != 1 {
		t.Fatalf("sent %d notifications, want exactly 1 across two checks", len(*log))
	}
	if got := (*log)[0].summary; !contains(got, "Widget - cache expires in") {
		t.Errorf("summary = %q, want it to name the project and the countdown", got)
	}
}

func TestWarnerRearmsAfterNewActivity(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	const id = "aaaa1111-0000-0000-0000-000000000000"
	seedSession(t, dir, "Widget", id, 150000, 53*time.Minute)

	log := stubSend(t)
	w := newWarner()
	w.check(time.Now())

	seedSession(t, dir, "Widget", id, 150000, 52*time.Minute)
	w.check(time.Now())

	if len(*log) != 2 {
		t.Fatalf("sent %d notifications, want 2 - a new message resets the expiry", len(*log))
	}
}

func TestWarnerStaysQuiet(t *testing.T) {
	cases := []struct {
		name   string
		cached int
		age    time.Duration
	}{
		{"cache too small to be worth re-warming", 5000, 53 * time.Minute},
		{"still far from expiry", 150000, 10 * time.Minute},
		{"already lapsed", 150000, 90 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", dir)
			seedSession(t, dir, "Widget", "aaaa1111-0000-0000-0000-000000000000", tc.cached, tc.age)

			log := stubSend(t)
			newWarner().check(time.Now())
			if len(*log) != 0 {
				t.Errorf("sent %d notifications, want silence", len(*log))
			}
		})
	}
}

func TestWarnTextTellsYouWhatToDo(t *testing.T) {
	_, closedBody := warnText(mkSession("Widget", 150000, false), 7*time.Minute)
	_, openBody := warnText(mkSession("Widget", 150000, true), 7*time.Minute)

	if !contains(openBody, "Send a message there") {
		t.Errorf("running session body = %q, want it to say to send a message", openBody)
	}
	if !contains(closedBody, "Resume it") {
		t.Errorf("closed session body = %q, want it to say to resume", closedBody)
	}
	if !contains(openBody, "150,000") {
		t.Errorf("body = %q, want the token count spelled out", openBody)
	}
}

func mkSession(project string, cached int, running bool) claude.Session {
	return claude.Session{Project: project, Cached: cached, Running: running}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
