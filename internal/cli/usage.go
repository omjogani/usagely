package cli

func usage() string {
	return bold("Usagely") + " - Claude Code usage limits in your system tray\n\n" +
		bold("Usage:") + "\n" +
		"  usagely [command]\n\n" +
		bold("Commands:") + "\n" +
		"  " + cyan("(none)     ") + " run the tray indicator\n" +
		"  " + cyan("status     ") + " print what the tray is showing right now\n" +
		"  " + cyan("install    ") + " add autostart and the Claude Code status line hook\n" +
		"  " + cyan("uninstall  ") + " undo install, restoring your own status line\n" +
		"  " + cyan("upgrade    ") + " fetch and build the latest release (needs Go)\n" +
		"  " + cyan("hook       ") + " capture usage from status line JSON on stdin\n" +
		"              " + faint("--wrap CMD   run CMD with the same payload afterwards") + "\n\n" +
		faint(`Claude Code runs "usagely hook" as your status line. It writes the rate limits
to ~/.cache/usagely.json and the tray reads them back - no credentials, no
network. Requires a Claude Pro or Max subscription.`) + "\n\n" +
		yellow("★") + " Saved you a trip to /usage? " +
		cyan("https://github.com/omjogani/Usagely") + faint(" - a star keeps it going") + "\n"
}
