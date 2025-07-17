package network

import (
	"net"
	"strings"
	"time"
)

func CheckSSH(host string) bool {
	address := net.JoinHostPort(host, "22")
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		// If there's an error (e.g., connection refused, timeout), the port is not open.
		return false
	}
	_ = conn.Close()
	return true
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