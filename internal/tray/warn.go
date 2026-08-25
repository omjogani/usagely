package tray

import (
	"fmt"
	"time"

	"github.com/omjogani/usagely/internal/claude"
	"github.com/omjogani/usagely/internal/notify"
)

const (
	warnLead      = 10 * time.Minute
	warnMinTokens = 20000
)

var send = notify.Send

type warner struct{ sent map[string]time.Time }

func newWarner() *warner { return &warner{sent: map[string]time.Time{}} }

func (w *warner) check(now time.Time) {
	for _, session := range claude.Sessions(now) {
		left := session.ExpiresIn(now)
		if left <= 0 || left > warnLead || session.Cached < warnMinTokens {
			continue
		}
		expiry := session.ExpiresAt()
		if w.sent[session.ID].Equal(expiry) {
			continue
		}
		w.sent[session.ID] = expiry

		summary, body := warnText(session, left)
		_ = send(summary, body)
	}
}

func warnText(session claude.Session, left time.Duration) (summary, body string) {
	summary = fmt.Sprintf("%s - cache expires in %s", session.Project, HumanDur(left))
	verb := "Send a message there"
	if !session.Running {
		verb = "Resume it"
	}
	body = fmt.Sprintf("%s tokens cached. %s to keep the cache warm - letting it lapse re-pays the whole context at 20x the read price.",
		thousands(session.Cached), verb)
	return summary, body
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
