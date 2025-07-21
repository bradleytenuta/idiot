package network

import (
	"io"
	stdlog "log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/rs/zerolog/log"

	"com.bradleytenuta/idiot/internal/model"
)

// PerformMdnsScan discovers services on the local network using mDNS.
// It queries for all available services and processes the results concurrently.
func PerformMdnsScan(iface *net.Interface, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	// A buffered channel is used to receive service entries from the mDNS query.
	mdnsEntries := make(chan *mdns.ServiceEntry, 100)
	var wg sync.WaitGroup

	// Goroutine to query for mDNS services.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(mdnsEntries)

		// Set up mDNS query parameters. We search for the special "_services._dns-sd._udp" name to discover all available services.
		params := mdns.DefaultParams("_services._dns-sd._udp")
		params.Timeout = 2 * time.Second
		params.Entries = mdnsEntries
		params.DisableIPv6 = true                     // We get IPv6 from the entry itself if available.
		params.Logger = stdlog.New(io.Discard, "", 0) // Suppress mdns library's default logger.

		if iface != nil {
			params.Interface = iface
		}

		if err := mdns.Query(params); err != nil {
			log.Debug().Msgf("mDNS query error: %v", err)
		}
	}()

	// Goroutine to process the mDNS entries as they are discovered.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range mdnsEntries {
			processMdnsEntry(entry, discoveredDevices, mu)
		}
	}()

	wg.Wait()
}

// processMdnsEntry handles a single discovered mDNS service. It extracts relevant
// information like IP addresses and hostname, and then safely updates the shared
// map of discovered devices.
func processMdnsEntry(entry *mdns.ServiceEntry, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	if entry.AddrV4 == nil {
		return
	}

	ipStr := entry.AddrV4.String()
	modelName := extractModelName(entry)
	var addrV6Str string
	if entry.AddrV6 != nil {
		addrV6Str = entry.AddrV6.String()
	}

	mu.Lock()
	defer mu.Unlock()

	// Get or create the device entry.
	device, exists := discoveredDevices[ipStr]
	if !exists {
		device = &model.Device{AddrV4: ipStr}
		discoveredDevices[ipStr] = device
	}

	// Update fields only if they are currently empty to avoid overwriting data from other sources.
	if device.Hostname == "" && modelName != "" {
		device.Hostname = modelName
	}
	if device.AddrV6 == "" && addrV6Str != "" {
		device.AddrV6 = addrV6Str
	}

	device.AddSource("mDNS")
}

// Searches for a model name (e.g., "md=Google Nest Mini")
// in the InfoFields of an mDNS entry using an idiomatic prefix check.
func extractModelName(entry *mdns.ServiceEntry) string {
	for _, field := range entry.InfoFields {
		if modelName, found := strings.CutPrefix(field, "md="); found {
			return modelName
		}
	}
	return ""
}
