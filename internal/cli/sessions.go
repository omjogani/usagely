package cli

import (
	"fmt"
	"time"

	"github.com/omjogani/usagely/internal/claude"
	"github.com/omjogani/usagely/internal/tray"
)

func runSessions() error {
	now := time.Now()
	sessions := claude.Sessions(now)

	fmt.Println(bold("Claude Code prompt caches"))
	fmt.Println(faint(statusRule))
	if len(sessions) == 0 {
		fmt.Println(faint("no warm caches - every session is idle past its TTL"))
		return nil
	}

	for _, session := range sessions {
		left := session.ExpiresIn(now)
		state := faint("closed ")
		if session.Running {
			state = green("running")
		}
		paint := green
		switch {
		case left <= 5*time.Minute:
			paint = red
		case left <= 15*time.Minute:
			paint = yellow
		}
		fmt.Printf("%s  %-24s %s %s\n",
			state, truncate(session.Project, 24),
			paint(fmt.Sprintf("%9s left", tray.HumanDur(left))),
			faint(fmt.Sprintf("%s cached", thousands(session.Cached))))
	}

	fmt.Println(faint(statusRule))
	fmt.Println(faint("resume within the window to keep reading the cache at 0.1x input price"))
	return nil
}

func truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	return s[:width-1] + "…"
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
