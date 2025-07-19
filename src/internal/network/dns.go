package network

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"com.bradleytenuta/idiot/internal/model"
)

// PerformReverseDnsLookUp enriches device data with hostnames found via reverse DNS lookups.
func PerformReverseDnsLookUp(discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	var wg sync.WaitGroup
	for _, device := range discoveredDevices {
		wg.Add(1)
		go func(deviceToProcess *model.Device) {
			defer wg.Done()

			if deviceToProcess.Hostname != "" {
				return
			}

			hostname := lookupHostname(deviceToProcess.AddrV4)

			if hostname != "" {
				mu.Lock()
				deviceToProcess.Hostname = hostname
				mu.Unlock()
			}
		}(device)
	}
	wg.Wait()
}

// lookupHostname performs a reverse DNS lookup for a given IP address.
func lookupHostname(ipStr string) string {
	// We use a context with a timeout to avoid waiting too long for a non-responsive lookup.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	hostnames, err := net.DefaultResolver.LookupAddr(ctx, ipStr)
	if err == nil && len(hostnames) > 0 {
		// We trim the trailing dot that FQDNs often have. e.g. "my-pc.lan." -> "my-pc.lan"
		fqdn := strings.TrimSuffix(hostnames[0], ".")
		// For local networks, we often get "hostname.lan" or "hostname.local".
		// We'll take the first part of the domain name as the simple hostname.
		hostname := strings.Split(fqdn, ".")[0]
		log.Debug().Msgf("Resolved %s -> %s (from FQDN: %s)", ipStr, hostname, fqdn)
		return hostname
	}

	log.Debug().Msgf("Could not resolve hostname for: %s", ipStr)
	return ""
}
