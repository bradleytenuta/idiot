package cmd

import (
  "fmt"
  "time"
  "sync"
  "os"
  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  "com.bradleytenuta/idiot/internal/model"
  "com.bradleytenuta/idiot/internal/network"
)

// runScan is the main execution function for the scan command.
// It orchestrates the network setup, discovery phases (mDNS, ICMP),
// post-processing (SSH checks), and final reporting.
func runScan(cmd *cobra.Command, args []string) {
  // --- Network Setup ---
  networkAddr, broadcastAddr, iface, err := network.GetInternetFacingNetworkInfo()
  if err != nil {
    fmt.Printf("Error setting up network: %v\n", err)
    return
  }

  // --- Concurrent Data Storage Setup ---
  // Create a map to store discovered devices, using the IP address string as the key for quick lookups.
  discoveredDevices := make(map[string]*model.Device)
  // A Mutex is used to prevent race conditions when multiple goroutines write to the map concurrently.
  var mu sync.Mutex

  // --- Discovery Phase 1: mDNS (Service Discovery) ---
  network.PerformMdnsScan(iface, discoveredDevices, &mu)

  // --- Discovery Phase 2: ICMP Scan (Host Discovery) ---
  network.PerformIcmpScan(networkAddr, broadcastAddr, discoveredDevices, &mu)

  // --- Post-Scan Processing ---
  // Wait a bit longer to ensure all mDNS responses have been processed.
  // TODO: A more robust solution would use a context with a timeout for the entire scan operation.
  time.Sleep(7 * time.Second)

  // TODO: 192.168.86.21 = Raspberry Pi 4
  // --- SSH Check Phase ---
  // Concurrently check for SSH availability on all discovered devices.
  var sshWg sync.WaitGroup
  for _, dev := range discoveredDevices {
    sshWg.Add(1)
    // Launch a goroutine for each device to check for SSH.
    go func(d *model.Device) {
      defer sshWg.Done() // This ensures that the WaitGroup's counter is decremented when the goroutine finishes, regardless of how it exits.
      // For each device, check if the SSH port is open and update its status.
      d.CanConnectSSH = network.CheckSSH(d.AddrV4.String())
    }(dev) // Pass the current device pointer to the goroutine to avoid closure issues.
  }
  sshWg.Wait() // Wait for all SSH checks to complete.

  // --- Discovery Phase 3: reverse DNS (rDNS) lookup. This asks a DNS server, "What hostname corresponds to this IP address?" ---
  network.ProcessDiscoveredIPs(discoveredDevices, &mu)

  // --- Final Report ---
  // Initialize a console-friendly logger for structured, readable output.
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}).With().Timestamp().Logger()
	log.Info().Msg("--- Discovered Devices ---")
	// Iterate through the map of discovered devices and log their details using the custom marshaler.
	for _, dev := range discoveredDevices {
    // .Send() is a performant way to dispatch the log event.
    log.Info().Object("device", dev).Send()
  }
}

// init function registers the scan command with the root command.
func init() {
  rootCmd.AddCommand(scanCmd)
}

// scanCmd defines the 'scan' command, its flags, and the main execution logic.
var scanCmd = &cobra.Command{
  Use:    "scan",
  Short:  "Scan the local network of this device and list the IP Addresses of devices connected to it.",
  Long:   `Scan the local network of this device and list the IP Addresses of devices connected to it. Including IPv4, IPv6 and ports reachable.`,
  Run:    runScan,
}

// Important note on ARP cache:
// After a successful ping, the device's MAC address should be in your system's ARP cache.
// Retrieving this from Go directly is OS-dependent and often involves:
// - Parsing `arp -a` output (less ideal for programmatic use)
// - Using a CGO binding to low-level network functions
// - Reading `/proc/net/arp` on Linux systems.
// The Go standard library does not provide a direct way to query the ARP cache.
// For a production-grade network scanner, you'd likely integrate with a library
// that wraps these OS-specific calls or use CGO.