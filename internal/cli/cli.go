package cli

import (
	"fmt"
	"os"

	"github.com/omjogani/usagely/internal/tray"
)

// Main dispatches os.Args and is the whole of the binary at the repo root.
func Main() {
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
	case "upgrade":
		exit(runUpgrade())
	case "-h", "--help", "help":
		fmt.Print(usage())
	default:
		fmt.Fprintf(os.Stderr, "usagely: unknown command %q\n\n%s", command, usage())
		os.Exit(2)
	}
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "usagely:", err)
		os.Exit(1)
	}
}
