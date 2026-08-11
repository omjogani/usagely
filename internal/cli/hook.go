package cli

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
)

func runHook(args []string) {
	flags := flag.NewFlagSet("hook", flag.ExitOnError)
	wrapped := flags.String("wrap", "", "status line command to run after capturing usage")
	_ = flags.Parse(args)

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		input = nil
	}

	dumpPayloadIfDebugFileExists(input)

	// Capturing is best-effort on purpose. This runs on every status line refresh
	if snapshot, ok := claude.Parse(input, time.Now()); ok {
		_ = claude.Write(snapshot)
	}

	if *wrapped != "" {
		passThrough(*wrapped, input)
		return
	}

	// Nothing was wrapped, so we are the whole status line.
	if snapshot, err := claude.Read(); err == nil {
		if line := summarise(snapshot, time.Now()); line != "" {
			fmt.Println(line)
		}
	}
}

func dumpPayloadIfDebugFileExists(input []byte) {
	debugFlagFile := claude.CachePath() + ".debug"
	if _, err := os.Stat(debugFlagFile); err != nil {
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

// summarise is the one-line form the hook prints when it is the whole status line
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
