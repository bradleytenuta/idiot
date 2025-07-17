//go:build windows

// TODO: This actually worked so we need to keep this and the terminal_other.go file.
package cmd

import (
	"os"

	"golang.org/x/sys/windows"
)

// initTerminal enables virtual terminal processing on Windows and returns a function to restore the original state.
func initTerminal() (cleanup func()) {
	// Start with a no-op cleanup function.
	cleanup = func() {}

	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32
	if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
		// Not a console or another error, we can't do anything.
		return
	}

	newMode := originalMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(stdout, newMode); err != nil {
		// Failed to set the mode, we can't do anything.
		return
	}

	// Return a function that will restore the original console mode.
	cleanup = func() {
		_ = windows.SetConsoleMode(stdout, originalMode)
	}
	return
}

