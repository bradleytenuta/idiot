package network

import (
  "fmt"
  "net"
  "strings"
  "time"
  "sync"
  "github.com/hashicorp/mdns"
  "com.bradleytenuta/idiot/internal/model"
)

// Starts the mDNS discovery process.
// It runs in the background, populating the shared discoveredDevices map with services it finds.
func PerformMdnsScan(iface *net.Interface, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
  // This goroutine will discover devices on the network that advertise services via multicast DNS.
  // A buffered channel is used to receive service entries from the mDNS query.
  mdnsEntries := make(chan *mdns.ServiceEntry, 100) // Buffer the channel
  go func() {
    // Ensure the channel is closed when the goroutine finishes to signal completion.
    defer close(mdnsEntries)
    // Set up mDNS query parameters. We search for the special "_services._dns-sd._udp" name to discover all available services.
    params := mdns.DefaultParams("_services._dns-sd._udp")
    params.Timeout = 5 * time.Second
    params.Entries = mdnsEntries
    params.DisableIPv6 = true
    // If a specific network interface was found, bind the mDNS query to it.
    if iface != nil {
      params.Interface = iface
    }
    // Execute the mDNS query.
    err := mdns.Query(params)
    if err != nil {
      fmt.Printf("mDNS query error: %v\n", err)
    }
  }()

  // This goroutine processes the results from the mDNS discovery channel.
  go func() {
    // Range over the channel until it's closed by the sender goroutine.
    for entry := range mdnsEntries {
      // Lock the mutex to safely access the shared map.
      mu.Lock()
      ipStr := entry.AddrV4.String()

      // Extract model name from mDNS entry info fields to get a user-friendly name.
			var modelName string
			for _, field := range entry.InfoFields {
        
				if strings.HasPrefix(field, "md=") {
					// As an example, split: "md=Google Nest Mini" into ["md", "Google Nest Mini"]
					parts := strings.SplitN(field, "=", 2)
					if len(parts) == 2 {
						modelName = parts[1]
						break // Found the model name, no need to check other fields.
					}
				}
			}

      // If the device hasn't been seen before, create a new entry in the map.
      if _, exists := discoveredDevices[ipStr]; !exists {
        discoveredDevices[ipStr] = &model.Device{AddrV4: entry.AddrV4, AddrV6: entry.AddrV6IPAddr, Hostname: modelName}
      } else {
        if discoveredDevices[ipStr].Hostname == "" {
          discoveredDevices[ipStr].Hostname = modelName
        }
        if discoveredDevices[ipStr].AddrV6 == nil {
          discoveredDevices[ipStr].AddrV6 = entry.AddrV6IPAddr
        }
      }

      discoveredDevices[ipStr].AddSource("mDNS")
      // Append the discovered service to the device's list of services. There can be multiple services per device.
      discoveredDevices[ipStr].Services = append(discoveredDevices[ipStr].Services, entry)
      // Unlock the mutex after modification.
      mu.Unlock()
    }
  }()
}