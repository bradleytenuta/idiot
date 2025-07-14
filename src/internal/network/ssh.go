package network

import (
  "net"
  "time"
)

// checkSSH attempts to connect to the SSH port (22) on a given host.
// It returns true if a TCP connection is successful, false otherwise.
func CheckSSH(host string) bool {
	// Combine the host IP with the standard SSH port number.
	address := net.JoinHostPort(host, "22")
	// Attempt to establish a TCP connection with a short timeout to avoid long waits.
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false // If there's an error (e.g., connection refused, timeout), the port is not open.
	}
	// It's good practice to close the connection immediately if we only care about reachability.
	_ = conn.Close()
	return true
}