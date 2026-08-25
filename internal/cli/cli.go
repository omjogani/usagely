package cli

import (
	"fmt"
	"os"

	"github.com/omjogani/usagely/internal/tray"
)

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
	case "sessions":
		exit(runSessions())
	case "install":
		exit(runInstall())
	case "uninstall":
		exit(runUninstall())
	case "upgrade":
		exit(runUpgrade())
	case "version", "-v", "--version":
		fmt.Println("usagely", version())
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
