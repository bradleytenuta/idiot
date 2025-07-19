package cmd

import (
  "bytes"
  "testing"
)

func TestVersionCmd(t *testing.T) {
  // Create a new buffer to capture the command's output.
  // This buffer will be used as the command's output stream.
  buf := new(bytes.Buffer)

  // The output is written from the root command's context.
  rootCmd.SetOut(buf)
  // Set the arguments for the root command to run the "version" subcommand.
  // This prevents Cobra from parsing os.Args, which contain test flags.
  rootCmd.SetArgs([]string{"version"})

  // Execute the root command. It will find and run the version subcommand.
  err := rootCmd.Execute()
  if err != nil {
    t.Fatalf("rootCmd.Execute() with 'version' arg failed with %v", err)
  }

  // Get the output from the buffer as a string.
  got := buf.String()
  // Define the expected output. Note that cmd.Println adds a newline character.
  expected := "1.0.0\n"

  if got != expected {
    t.Errorf("unexpected version output.\ngot:  %q\nwant: %q", got, expected)
  }
}