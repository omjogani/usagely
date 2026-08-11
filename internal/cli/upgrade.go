package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const modulePath = "github.com/omjogani/usagely"

func runUpgrade() error {
	fmt.Println("upgrading:     go install " + modulePath + "@latest")

	cmd := exec.Command("go", "install", modulePath+"@latest")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("go toolchain not found on PATH - install Go, or rebuild from source")
		}
		return fmt.Errorf("go install: %w", err)
	}

	installed := goBinPath()
	fmt.Println("upgraded:     ", installed)
	if shadowed(installed) {
		fmt.Printf("\n%s the binary you are running is %s, not the one just upgraded.\n"+
			"Replace it, or drop %s earlier on your PATH.\n",
			yellow("warning:"), currentExe(), filepath.Dir(installed))
	}
	fmt.Println("\nRestart the tray to pick it up:  pkill usagely; usagely &")
	return nil
}

// goBinPath is where "go install" just put the binary: GOBIN, or GOPATH/bin.
func goBinPath() string {
	out, err := exec.Command("go", "env", "GOBIN", "GOPATH").Output()
	if err != nil {
		return "usagely"
	}
	return parseGoBin(string(out))
}

// parseGoBin turns the two lines of "go env GOBIN GOPATH" into a binary path.
// GOBIN wins when set; otherwise it is GOPATH/bin.
func parseGoBin(out string) string {
	env := strings.Split(strings.TrimRight(out, "\n"), "\n")
	dir := strings.TrimSpace(env[0])
	if dir == "" {
		if len(env) < 2 || strings.TrimSpace(env[1]) == "" {
			return "usagely"
		}
		dir = filepath.Join(strings.TrimSpace(env[1]), "bin")
	}
	return filepath.Join(dir, "usagely")
}

// shadowed reports whether the running binary is somewhere else, in which case
// the upgrade lands in a file the user is not actually running.
func shadowed(installed string) bool {
	exe := currentExe()
	if exe == "" || installed == "usagely" {
		return false
	}
	return exe != installed
}

func currentExe() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}
