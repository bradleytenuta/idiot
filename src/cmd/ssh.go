package cmd

import (
  "fmt"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"
  "log"
  "os"
  "golang.org/x/crypto/ssh"
  "golang.org/x/term"
  "com.bradleytenuta/idiot/internal/model"
  "com.bradleytenuta/idiot/internal/network"
  "com.bradleytenuta/idiot/internal/ui"
)

// init function registers the scan command with the root command.
func init() {
  rootCmd.AddCommand(sshCmd)
}

// scanCmd defines the 'scan' command, its flags, and the main execution logic.
var sshCmd = &cobra.Command{
  Use:    "ssh",
  Short:  "",
  Long:   ``,
  Run:    runSsh,
}

// GetSelectedDevicesFromConfig reads the configuration and returns a slice of fully-formed *model.Device.
func getSelectedDevicesFromConfig() ([]model.Device, error) {
	// Create a slice to hold the data from the config file.
	var configDevices []model.Device

	// Unmarshal the config data directly into our slice of serializable structs.
	if err := viper.UnmarshalKey("selected_devices", &configDevices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal 'selected_devices' from config: %w", err)
	}

	return configDevices, nil
}

// TODO: This looks terrible in windows CMD.
func runSsh(cmd *cobra.Command, args []string) {
  // On Windows, this enables virtual terminal processing, which is required
  // for both promptui and the SSH session's ANSI escape codes to work correctly.
  // It returns a cleanup function that is deferred to restore the terminal state.
  defer initTerminal()()

  savedDevices, _ := getSelectedDevicesFromConfig()
  // We will create a map where the key is the device's Name.
  deviceMap := make(map[string]*model.Device)
  // Iterate over the slice by index. This is crucial for getting
  // a correct and stable pointer to each element.
  for i := range savedDevices {
      // Get a pointer to the device at the current index.
      device := &savedDevices[i]
      // Use the device's Name as the key in the new map.
      deviceMap[device.AddrV4] = device
  }

  selectedDevice := ui.CreateInteractiveSelect(deviceMap)
  
  user := ui.GetPromptInput("Username", 0)
  password := ui.GetPromptInput("Password", '*')
  
  // replace with select from saved IPs.
  addr := selectedDevice.AddrV4
  addr = network.AddSshPortIfMissing(addr)

	// 3. Configure the SSH client
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// IMPORTANT: In a real-world application, you should use a more secure
		// HostKeyCallback, like ssh.FixedHostKey or one that checks a known_hosts file.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 4. Establish the SSH connection
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer client.Close()
	fmt.Printf("Connected to %s\n", addr)

	// 5. Create a new session
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	// 6. Set up the interactive terminal
	// Get file descriptors for stdin and stdout
	inFd := int(os.Stdin.Fd())
	outFd := int(os.Stdout.Fd())

	// Check if the session is running in an interactive terminal
	if !term.IsTerminal(inFd) || !term.IsTerminal(outFd) {
		log.Fatalf("Cannot create an interactive SSH session: Standard I/O is not a terminal.")
	}

	// Put the local terminal into raw mode
	oldState, err := term.MakeRaw(inFd)
	if err != nil {
		log.Fatalf("Failed to put terminal in raw mode: %v", err)
	}
	defer term.Restore(inFd, oldState)

	// Get terminal dimensions and request a PTY from the remote server
	width, height, err := term.GetSize(outFd)
	if err != nil {
		log.Fatalf("Failed to get terminal size: %v", err)
	}

	// Request a pseudo-terminal (pty) with the correct dimensions
	if err := session.RequestPty("xterm-256color", height, width, ssh.TerminalModes{}); err != nil {
		log.Fatalf("Request for pty failed: %v", err)
	}

	// 7. Connect the local terminal I/O to the remote session
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// 8. Start the remote shell
	if err := session.Shell(); err != nil {
		log.Fatalf("Failed to start shell: %v", err)
	}

	// 9. Wait for the session to finish (user logs out)
	// The error is ignored because it's typically "wait: remote command exited"
	// when the user types `exit`.
	_ = session.Wait()
}