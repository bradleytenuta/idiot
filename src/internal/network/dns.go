package network

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"time"
	"com.bradleytenuta/idiot/internal/model"
)

// enrichDeviceData takes a device with an IP and adds more information, like a hostname.
func enrichDeviceData(device *model.Device) *model.Device {
	ipStr := device.AddrV4.String()

	// Perform a reverse DNS lookup to get the hostname.
	// We use a context with a timeout to avoid waiting too long for a non-responsive lookup.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use the context-aware resolver for the lookup.
	if (device.Hostname == "") {
		hostnames, err := net.DefaultResolver.LookupAddr(ctx, ipStr)
		if err == nil && len(hostnames) > 0 {
			// Success! Use the first result.
			// We trim the trailing dot that FQDNs often have. e.g. "my-pc.lan." -> "my-pc.lan"
			fqdn := strings.TrimSuffix(hostnames[0], ".")
			// For local networks, we often get "hostname.lan" or "hostname.local".
			// We'll take the first part of the domain name as the simple hostname.
			device.Hostname = strings.Split(fqdn, ".")[0]
			log.Printf("Resolved %s -> %s (from FQDN: %s)", ipStr, device.Hostname, fqdn)
		} else {
			// This is a common case; many IPs on a local network won't have a PTR record.
			log.Printf("Could not resolve hostname for %s", ipStr)
		}
	} else {
		log.Printf("Skipping hostname resolution for %s", ipStr)
	}

	return device
}

// In your main scanning logic, after you get a successful ICMP reply for an IP:
func PerformReverseDnsLookUp(discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	// A WaitGroup is used to wait for all the concurrent ping operations to complete.
  	var wg sync.WaitGroup

	// Enrich the basic IP with more details like hostname.
	for ip4, device := range discoveredDevices {
		wg.Add(1)
		go func(ipToProcess string, deviceToProcess *model.Device) {
			defer wg.Done()
			mu.Lock()
			deviceToProcess = enrichDeviceData(deviceToProcess)
			discoveredDevices[ipToProcess] = deviceToProcess
			mu.Unlock()
		}(ip4, device)
	}
	wg.Wait()
}