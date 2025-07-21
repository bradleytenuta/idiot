package network

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"com.bradleytenuta/idiot/internal/model"
)

// PerformSSHScan checks if SSH is available on the discovered devices.
func PerformSSHScan(discoveredDevices map[string]*model.Device) {
	var sshWg sync.WaitGroup
	for _, dev := range discoveredDevices {
		sshWg.Add(1)
		// Launch a goroutine for each device to check for SSH.
		go func(d *model.Device) {
			defer sshWg.Done() // This ensures that the WaitGroup's counter is decremented when the goroutine finishes, regardless of how it exits.
			// For each device, check if the SSH port is open and update its status.
			d.CanConnectSSH = checkSSH(d.AddrV4)
		}(dev) // Pass the current device pointer to the goroutine to avoid closure issues.
	}
	sshWg.Wait() // Wait for all SSH checks to complete.
}

// AddPort ensures that an address string has a port. If the port is missing,
// it appends the default SSH port "22". It returns an error if the address
// is malformed in a way other than a missing port.
func AddPort(addr string) (string, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port") {
			return net.JoinHostPort(addr, "22"), nil
		}
	}
	return "", err
}

// GetHostKeyCallback creates a callback function that verifies server host keys
// against the user's known_hosts file (e.g., ~/.ssh/known_hosts).
// This is the recommended secure approach to prevent man-in-the-middle attacks.
func GetHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// knownhosts.New will create the file if it doesn't exist.
	// It returns a callback that verifies the host key. When you connect to an SSH server, it presents 
	// a unique cryptographic "host key" to identify itself. Your SSH client's job is to verify that 
	// this key is the correct one for the server you think you're connecting to.
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create known_hosts callback from '%s': %w", knownHostsPath, err)
	}
	return callback, nil
}

// checkSSH performs a quick check to see if a TCP connection can be established
// to port 22 on the given host. It uses a short timeout to avoid long waits.
func checkSSH(host string) bool {
	address := net.JoinHostPort(host, "22")
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		// If there's an error (e.g., connection refused, timeout), the port is not open.
		return false
	}
	_ = conn.Close()
	return true
}
