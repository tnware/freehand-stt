package app

// StartupRequested reports whether the process was launched by the Windows
// startup entry rather than by the user. The match is exact: a value-carrying
// form such as --startup=true is deliberately not accepted, so that a crafted
// argument cannot suppress the settings window.
func StartupRequested(args []string) bool {
	if len(args) < 2 {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "--startup" {
			return true
		}
	}
	return false
}
