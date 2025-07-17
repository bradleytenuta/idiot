//go:build windows

package ui

import (
	"os"
	"golang.org/x/sys/windows"
	"github.com/rs/zerolog/log"
)

// This function is built for windows applications only
// This function is to enable ANSI/VT100 escape code processing in the Windows console
func InitTerminal() (cleanup func()) {
	// Start with a no-op cleanup function
	cleanup = func() {}

	// Gets a handle to the standard output console (os.Stdout)
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	// Calls the Windows API function GetConsoleMode to get the current configuration of the console and saves it in the originalMode variable
	if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
		log.Debug().Msgf("Error getting console mode: %v", err)
		return
	}

	// Creates a newMode by taking the originalMode and adding the windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING flag
	// This is the key step that tells the console to start interpreting ANSI escape codes
	newMode := originalMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

	// It applies this newMode to the console using SetConsoleMode.
	if err := windows.SetConsoleMode(stdout, newMode); err != nil {
		log.Debug().Msgf("Error setting console mode: %v", err)
		return
	}

	// If successful, it returns a cleanup function. The calling code is expected to execute this function 
	// when it's finished (typically using defer). This cleanup function restores the console back to its originalMode, 
	// ensuring the application doesn't permanently alter the user's terminal settings after it exits.
	cleanup = func() {
		_ = windows.SetConsoleMode(stdout, originalMode)
	}

	return
}