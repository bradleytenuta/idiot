//go:build !windows

package cmd

// initTerminal is a no-op on non-Windows platforms. It returns a no-op cleanup function.
func initTerminal() func() {
	return func() {}
}

