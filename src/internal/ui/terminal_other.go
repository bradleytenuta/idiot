//go:build !windows

package ui

// This is a no operation function that is built into the application for non-Windows systems.
// This is handled by the build tag at the top.
func InitTerminal() func() {
	return func() {}
}