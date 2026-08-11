// Command usagely shows Claude Code's rate-limit usage in the desktop tray.
//
//	usagely             run the tray indicator
//	usagely status      print what the tray is showing right now
//	usagely install     add autostart and the Claude Code status line hook
//	usagely uninstall   undo install
//	usagely hook        capture usage from status line JSON on stdin
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/omjogani/usagely/internal/claude"
	"github.com/omjogani/usagely/internal/tray"
)

const usage = `usagely — Claude Code usage limits in your system tray

  usagely             run the tray indicator
  usagely status      print what the tray is showing right now
  usagely install     add autostart and the Claude Code status line hook
  usagely uninstall   undo install
  usagely hook        capture usage from status line JSON on stdin
`

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "", "tray":
		tray.Run()
	case "hook":
		runHook(os.Args[2:])
	case "status":
		exit(runStatus())
	case "install":
		exit(runInstall())
	case "uninstall":
		exit(runUninstall())
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "usagely: unknown command %q\n\n%s", command, usage)
		os.Exit(2)
	}
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "usagely:", err)
		os.Exit(1)
	}
}

// runHook stands in for the user's status line: it captures the rate limits
// out of the payload Claude Code sends, then hands the same payload to
// whatever command it replaced.
func runHook(args []string) {
	flags := flag.NewFlagSet("hook", flag.ExitOnError)
	wrapped := flags.String("wrap", "", "status line command to run after capturing usage")
	_ = flags.Parse(args)

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		input = nil
	}

	dumpPayload(input)

	// Capturing is best-effort on purpose. This runs on every status line
	// refresh, and a failed cache write must never break someone's prompt.
	if snapshot, ok := claude.Parse(input, time.Now()); ok {
		_ = claude.Write(snapshot)
	}

	if *wrapped != "" {
		passThrough(*wrapped, input)
		return
	}

	// Nothing was wrapped, so we are the whole status line. Print something
	// rather than leaving the user with an empty row.
	if snapshot, err := claude.Read(); err == nil {
		if line := summarise(snapshot, time.Now()); line != "" {
			fmt.Println(line)
		}
	}
}

// runStatus prints what the tray is showing right now, plus where it came
// from, so a wrong-looking indicator can be compared against /usage without
// guessing at which layer went wrong.
func runStatus() error {
	snapshot, err := claude.Read()
	if err != nil {
		return fmt.Errorf("no data yet (%s): run a Claude Code session first", claude.CachePath())
	}

	now := time.Now()
	rows, panel := tray.Render(snapshot, now)
	for _, row := range rows {
		fmt.Println(row)
	}
	if panel > 0 {
		fmt.Printf("\npanel shows   %.0f%%\n", panel)
	}
	fmt.Printf("source        %s\n", claude.CachePath())
	fmt.Printf("captured      %s (%s ago)\n",
		time.Unix(snapshot.UpdatedAt, 0).Format("15:04:05"),
		now.Sub(time.Unix(snapshot.UpdatedAt, 0)).Round(time.Second))
	return nil
}

// dumpPayload keeps a copy of the last status line payload when the user has
// created the flag file, so "the numbers look wrong" can be diagnosed against
// what Claude Code actually sent rather than against what we parsed out of it.
//
//	touch ~/.cache/usagely.json.debug   # then read ~/.cache/usagely.json.payload
//	rm ~/.cache/usagely.json.debug      # to stop
func dumpPayload(input []byte) {
	if _, err := os.Stat(claude.CachePath() + ".debug"); err != nil {
		return
	}
	_ = os.WriteFile(claude.CachePath()+".payload", input, 0o600)
}

func passThrough(command string, input []byte) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func summarise(s claude.Snapshot, now time.Time) string {
	var parts []string
	if pct, _, ok := s.FiveHour.Live(now); ok {
		parts = append(parts, fmt.Sprintf("5h %.0f%%", pct))
	}
	if pct, _, ok := s.SevenDay.Live(now); ok {
		parts = append(parts, fmt.Sprintf("7d %.0f%%", pct))
	}
	return strings.Join(parts, " · ")
}
