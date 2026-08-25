package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const usageURL = "https://api.anthropic.com/api/oauth/usage"

func CredentialsPath() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, ".credentials.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", ".credentials.json")
}

func accessToken(now time.Time) (string, error) {
	path := CredentialsPath()
	if path == "" {
		return "", errors.New("cannot locate Claude Code credentials")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var creds struct {
		OAuth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   int64  `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}

	if err := json.Unmarshal(b, &creds); err != nil {
		return "", err
	}

	if creds.OAuth.AccessToken == "" {
		return "", errors.New("no Claude Code login found")
	}

	if creds.OAuth.ExpiresAt > 0 && time.UnixMilli(creds.OAuth.ExpiresAt).Before(now) {
		return "", errors.New("access token expired: start Claude Code to renew it")
	}

	return creds.OAuth.AccessToken, nil
}

func Fetch(now time.Time) (Snapshot, error) {
	token, err := accessToken(now)
	if err != nil {
		return Snapshot{}, err
	}

	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return Snapshot{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Snapshot{}, fmt.Errorf("usage endpoint returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Snapshot{}, err
	}
	return ParseUsage(body, now)
}

type apiWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    *string `json:"resets_at"`
}

func (w *apiWindow) window() *Window {
	if w == nil {
		return nil
	}
	out := &Window{UsedPercent: w.Utilization}
	if w.ResetsAt != nil {
		if t, err := time.Parse(time.RFC3339, *w.ResetsAt); err == nil {
			out.ResetsAt = t.Unix()
		}
	}
	return out
}

func ParseUsage(body []byte, now time.Time) (Snapshot, error) {
	var payload struct {
		FiveHour *apiWindow `json:"five_hour"`
		SevenDay *apiWindow `json:"seven_day"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Snapshot{}, err
	}
	if payload.FiveHour == nil && payload.SevenDay == nil {
		return Snapshot{}, errors.New("usage response carried no rate-limit windows")
	}
	return Snapshot{
		FiveHour:  payload.FiveHour.window(),
		SevenDay:  payload.SevenDay.window(),
		UpdatedAt: now.Unix(),
	}, nil
}
