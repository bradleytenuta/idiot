package cmd
// TODO: This looks terrible in windows CMD.
import (
  "os"
  "github.com/spf13/cobra"
  "github.com/rs/zerolog/log"
  "golang.org/x/crypto/ssh"
  "golang.org/x/term"
  "com.bradleytenuta/idiot/internal/model"
  "com.bradleytenuta/idiot/internal/network"
  "com.bradleytenuta/idiot/internal/ui"
  "com.bradleytenuta/idiot/internal"
)

func init() {
  rootCmd.AddCommand(sshCmd)
}

var sshCmd = &cobra.Command{
  Use:    "ssh",
  Short:  "Select the IOT device to SSH into.",
  Long:   `Select one of the saved IOT devices from the scan command to SSH into.`,
  Run:    runSsh,
}

func runSsh(cmd *cobra.Command, args []string) {
  // We are calling a function that returns another function, and then deferring the execution of the returned function.
  // This uses the function returned by initTerminal  schedules it to be executed right before the surrounding function exits.
  defer ui.InitTerminal()()
  addr, user, password, err := getLoginDetails()
  if err != nil {
    return
  }
  client, err := getClient(addr, user, password)
  if err != nil {
    log.Error().Msgf("Failed to create client: %v", err)
    return
  }
  defer client.Close()
  session, err := client.NewSession()
  if err != nil {
    log.Error().Msgf("Failed to create session: %v", err)
    return
  }
	defer session.Close()
  handleInteractiveSession(session)
}

func getLoginDetails() (string, string, string, error) {
  savedDevices := internal.ReadIotDevices()

  // TODO: Could we replace the map with a slice?
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

  selectedDevice, err := ui.CreateInteractiveSelect(deviceMap, "Select an IOT Device to SSH into")
  if err != nil {
    return "", "", "", err
  }

  user, err := ui.GetPromptInput("Username", 0)
  if err != nil {
    log.Error().Msgf("Failed to get username: %v", err)
    return "", "", "", err
  }

  password, err := ui.GetPromptInput("Password", '*')
  if err != nil {
    log.Error().Msgf("Failed to get password: %v", err)
    return "", "", "", err
  }

  addr, err := network.AddPort(selectedDevice.AddrV4)
  if err != nil {
    log.Error().Msgf("Invalid address: %v", err)
    return "", "", "", err
  }
  return addr, user, password, nil
}

func getClient(addr string, user string, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// TODO: IMPORTANT: In a real-world application, you should use a more secure
		// HostKeyCallback, like ssh.FixedHostKey or one that checks a known_hosts file.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
    return nil, err
	}
  return client, nil
}

func handleInteractiveSession(session *ssh.Session) {
  // A file descriptor is a small, non-negative integer that a process uses to identify an open file or other I/O resource
  // Gets the file descriptor for standard input (os.Stdin). This will almost always be 0.
	inFd := int(os.Stdin.Fd())
  // Gets the file descriptor for standard output (os.Stdout). This will almost always be 1.
	outFd := int(os.Stdout.Fd())

	// Check if the session is running in an interactive terminal
	if !term.IsTerminal(inFd) || !term.IsTerminal(outFd) {
		log.Error().Msg("We cannot create an interactive SSH session if the Standard I/O is not a terminal!")
    return
	}

  // Putting the terminal into raw mode means switching it from its normal, line-by-line "cooked" 
  // mode to a state where your program receives every single keystroke exactly as it's typed, immediately.
  // This is need because SSH client needs to pass every keystroke directly to the remote server so that the remote shell
  // can handle things like command history (up arrow) and auto-completion (Tab). Your local terminal can't be allowed to interfere.
	oldState, err := term.MakeRaw(inFd)
	if err != nil {
		log.Error().Msgf("Failed to put terminal in raw mode: %v", err)
    return
	}
  // If the application crashes or exits without restoring the terminal's oldState, the user's shell will be left in raw mode.
	defer term.Restore(inFd, oldState)

	// Get terminal dimensions and request a PTY from the remote server
	width, height, err := term.GetSize(outFd)
	if err != nil {
		log.Error().Msgf("Failed to get terminal size: %v", err)
    return
	}

  // Creates a pseudo-terminal (pty) with the same dimensions as the local terminal. We
  // are using the existing terminal, but we are temporarily changing its behavioral mode for the duration of the SSH session.
	if err := session.RequestPty("xterm-256color", height, width, ssh.TerminalModes{}); err != nil {
		log.Error().Msgf("Request for pty failed: %v", err)
    return
	}

	// Connects the session's standard input, output, and error streams to the SSH session.
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Shell(); err != nil {
		log.Error().Msgf("Failed to start shell: %v", err)
    return
	}

  // Wait for the session to end, usually when the user types `exit`.
	session.Wait()
}