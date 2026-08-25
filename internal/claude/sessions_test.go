package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTranscript(t *testing.T, dir, project, id, body string, age time.Duration) string {
	t.Helper()
	projectDir := filepath.Join(dir, "projects", project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, id+".jsonl")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	return path
}

const oneHourTurn = `{"cwd":"/home/me/work/Widget"}
{"message":{"usage":{"cache_read_input_tokens":900,"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_1h_input_tokens":100,"ephemeral_5m_input_tokens":0}}}}
{"message":{"usage":{"cache_read_input_tokens":5000,"cache_creation_input_tokens":50,"cache_creation":{"ephemeral_1h_input_tokens":50,"ephemeral_5m_input_tokens":0}}}}
`

func TestSessionsReadsLatestTurn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	writeTranscript(t, dir, "-home-me-work-Widget", "aaaa1111-0000-0000-0000-000000000000", oneHourTurn, 20*time.Minute)

	sessions := Sessions(time.Now())
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]
	if s.Cached != 5050 {
		t.Errorf("cached = %d, want 5050 from the last turn", s.Cached)
	}
	if s.TTL != time.Hour {
		t.Errorf("ttl = %s, want 1h", s.TTL)
	}
	if s.Project != "Widget" {
		t.Errorf("project = %q, want Widget", s.Project)
	}
	if s.Running {
		t.Error("running = true, want false for a pid-less transcript")
	}
	if left := s.ExpiresIn(time.Now()); left < 39*time.Minute || left > 41*time.Minute {
		t.Errorf("expires in %s, want about 40m", left)
	}
}

func TestSessionsSkipsExpiredAndUncached(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)

	writeTranscript(t, dir, "-home-me-work-Old", "bbbb2222-0000-0000-0000-000000000000", oneHourTurn, 3*time.Hour)
	writeTranscript(t, dir, "-home-me-work-Plain", "cccc3333-0000-0000-0000-000000000000",
		`{"cwd":"/home/me/work/Plain"}
{"message":{"usage":{"input_tokens":40,"output_tokens":9}}}
`, time.Minute)

	if sessions := Sessions(time.Now()); len(sessions) != 0 {
		t.Fatalf("got %d sessions, want 0 (one lapsed, one never cached)", len(sessions))
	}
}

func TestSessionsPrefersFiveMinuteBucket(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	writeTranscript(t, dir, "-home-me-work-Short", "dddd4444-0000-0000-0000-000000000000",
		`{"cwd":"/home/me/work/Short"}
{"message":{"usage":{"cache_read_input_tokens":2000,"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":10}}}}
`, time.Minute)

	sessions := Sessions(time.Now())
	if len(sessions) != 1 || sessions[0].TTL != 5*time.Minute {
		t.Fatalf("got %+v, want a single session on the 5m bucket", sessions)
	}
}

func TestSessionsSortsBySoonestExpiry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	const late = "eeee5555-0000-0000-0000-000000000000"
	const soon = "ffff6666-0000-0000-0000-000000000000"
	writeTranscript(t, dir, "-home-me-work-Late", late, oneHourTurn, 5*time.Minute)
	writeTranscript(t, dir, "-home-me-work-Soon", soon, oneHourTurn, 50*time.Minute)

	sessions := Sessions(time.Now())
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].ID != soon || sessions[1].ID != late {
		t.Errorf("order = %s then %s, want the soonest to expire first", sessions[0].ID[:4], sessions[1].ID[:4])
	}
}
