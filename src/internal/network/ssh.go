package network

import (
	"net"
	"strings"
	"time"
  "sync"
  "com.bradleytenuta/idiot/internal/model"
)

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

func AddPort(addr string) (string, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port") {
			return net.JoinHostPort(addr, "22"), nil
		}
	}
	return "", err
}

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