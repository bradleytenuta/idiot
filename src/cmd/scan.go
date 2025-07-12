package cmd

import (
  "fmt"
  "net"
  "strings"
  "time"
  "sync"
  "os"
  "github.com/hashicorp/mdns"
  "golang.org/x/net/icmp"
  "golang.org/x/net/ipv4"
  "github.com/spf13/cobra"
  "github.com/rs/zerolog"
  "com.bradleytenuta/idiot/internal/model"
)

// runScan is the main execution function for the scan command.
// It orchestrates the network setup, discovery phases (mDNS, ICMP),
// post-processing (SSH checks), and final reporting.
func runScan(cmd *cobra.Command, args []string) {
  // --- Network Setup ---
  // TODO: replace with viper.Get("network_name")
  interfaceName := "Ethernet 2"
  networkAddr, broadcastAddr, iface, err := getNetworkInfo(interfaceName)
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
  performMdnsScan(iface, discoveredDevices, &mu)

  // --- Discovery Phase 2: ICMP Scan (Host Discovery) ---
  performIcmpScan(networkAddr, broadcastAddr, discoveredDevices, &mu)

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
      d.CanConnectSSH = checkSSH(d.AddrV4.String())
    }(dev) // Pass the current device pointer to the goroutine to avoid closure issues.
  }
  sshWg.Wait() // Wait for all SSH checks to complete.

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

// getNetworkInfo finds the specified network interface, its IPv4 details,
// and calculates the network and broadcast addresses. It also corrects a potential
// bug with capturing a loop variable's address.
func getNetworkInfo(interfaceName string) (net.IP, net.IP, *net.Interface, error) {
  // Get a list of all network interfaces on the host machine.
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting interfaces: %w", err)
	}

	var localIP net.IP
	var subnetMask net.IPMask
	var selectedIface *net.Interface // Pointer to the interface found

	// Iterate over all found network interfaces by index to safely get a pointer.
	for i := range ifaces {
		// Use a local variable for the current interface for readability.
		currentIface := ifaces[i]
		// Check if the current interface name contains the target name.
		if strings.Contains(currentIface.Name, interfaceName) {
			// Get all addresses associated with the current interface.
			addrs, err := currentIface.Addrs()
			if err != nil {
				// Log the error but continue, as there might be other matching interfaces.
				fmt.Printf("Warning: could not get addresses for %s: %v\n", currentIface.Name, err)
				continue
			}
			// Iterate over the addresses of the interface.
			for _, a := range addrs {
				// Check if the address is an IP network and not a loopback address.
				if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					// We are interested in IPv4 addresses for this scan.
					if ipNet.IP.To4() != nil {
						// Store the IPv4 address, subnet mask, and a pointer to the interface.
            // Even if IPv4 (length == 4), Go will store this in an IPv6 format with length == 16.
            // This the local IP address of the current device.
						localIP = ipNet.IP.To4()
						subnetMask = ipNet.Mask
						selectedIface = &ifaces[i] // Safely get the address of the slice element.
						break                      // Exit the address loop once a suitable IPv4 address is found.
					}
				}
			}
		}
		// If we've found our IP, we can stop searching through interfaces.
		if localIP != nil {
			break
		}
	}

	// If no suitable interface and IPv4 address were found, return an error.
	if localIP == nil {
		return nil, nil, nil, fmt.Errorf("could not find a suitable IPv4 address on an interface matching '%s'", interfaceName)
	}

	fmt.Printf("Found Local IP: %s/%s on interface: %s\n", localIP.String(), net.IP(subnetMask).String(), selectedIface.Name)

	// --- Subnet Calculation ---
  // Calculate the network address by applying the subnet mask to the local IP.
	networkAddr := localIP.Mask(subnetMask)
  // Prepare a slice to hold the broadcast address.
	broadcastAddr := make(net.IP, len(localIP))
	for i := 0; i < len(localIP); i++ {
    // networkAddr[i] | ... (Bitwise OR)
    // This takes the current byte of the networkAddr and performs a bitwise OR operation with the result of ^subnetMask[i].
    // When you OR the networkAddr byte with the ^subnetMask[i] byte:
    // For the network bits (where networkAddr[i] has the network portion and ^subnetMask[i] has 0s), the OR operation will preserve the networkAddr bits.
    // For the host bits (where networkAddr[i] has 0s and ^subnetMask[i] has 1s), the OR operation will set all these bits to 1.
    //
    // ^subnetMask[i] - (Bitwise NOT/Complement)
    // This takes the current byte of the subnetMask and flips all its bits. For example, if a byte of subnetMask is 255 (binary 11111111), ^255 would be 0 (binary 00000000).
    // The effect of ^subnetMask[i] is to create a byte where the network bits (which were 1s in the subnet mask) become 0s, 
    // and the host bits (which were 0s in the subnet mask) become 1s. This essentially gives you the "wildcard" or "host part" of the subnet mask.
		broadcastAddr[i] = networkAddr[i] | ^subnetMask[i]
	}
	fmt.Printf("Network Address: %s\n", networkAddr.String())
	fmt.Printf("Broadcast Address: %s\n", broadcastAddr.String())

	return networkAddr, broadcastAddr, selectedIface, nil
}

// performMdnsScan starts the mDNS discovery process.
// It runs in the background, populating the shared discoveredDevices map with services it finds.
func performMdnsScan(iface *net.Interface, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
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

// performIcmpScan scans the local network using ICMP echo requests (pings) to discover hosts.
// It iterates through the IP range defined by the network and broadcast addresses.
// For each responsive host, it adds or updates an entry in the shared discoveredDevices map.
func performIcmpScan(networkAddr, broadcastAddr net.IP, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
  // A WaitGroup is used to wait for all the concurrent ping operations to complete.
  var wg sync.WaitGroup
  // TODO: replace with viper.Get("subnet_size")
  ipRangeStart := int(networkAddr[3]) // Assuming /24 for simplicity, adjust for larger subnets. TODO: Be flexible based on what subnet is in property file.
  ipRangeEnd := int(broadcastAddr[3])

  // For subnets larger than /24, you'd need to iterate through the network part too.
  // For example, for a /16:
  // startIPBytes := networkAddr.To4()
  // endIPBytes := broadcastAddr.To4()
  // for i := startIPBytes[0]; i <= endIPBytes[0]; i++ {
  //    for j := startIPBytes[1]; j <= endIPBytes[1]; j++ {
  //        // ... and so on
  //    }
  // }

  // Loop through the host portion of the IP range, skipping the network and broadcast addresses.
  for i := ipRangeStart + 1; i < ipRangeEnd; i++ {
    targetIP := make(net.IP, 4)
    copy(targetIP, networkAddr)
    targetIP[3] = byte(i) // Only works for /24 and smaller ranges within the last octet

    // Increment the WaitGroup counter for each new goroutine.
    wg.Add(1)
    // Launch a goroutine to ping the target IP address concurrently.
    go func(ip net.IP) {
      // Decrement the WaitGroup counter when the goroutine completes.
      defer wg.Done()
      // Listen for ICMP packets on all available IPv4 interfaces.
      conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
      if err != nil {
        // Silently return on error to avoid cluttering output.
        return
      }
      defer conn.Close()

      // Construct an ICMP Echo Request (ping) message.
      wm := icmp.Message{
        Type: ipv4.ICMPTypeEcho, Code: 0,
        Body: &icmp.Echo{
          ID:   os.Getpid() & 0xffff,       // Use the process ID to uniquely identify this pinger.
          Seq:  1,                          // Sequence number.
          Data: []byte("HELLO-R-U-THERE"),
        },
      }

      // Marshal the message into its binary wire format.
      wb, err := wm.Marshal(nil)
      if err != nil {
        return
      }

      // Send the ICMP packet to the target IP address.
      _, err = conn.WriteTo(wb, &net.IPAddr{IP: ip})
      if err != nil {
        // Silently return on send error.
        return
      }

      // Set a deadline for reading a reply to avoid waiting indefinitely.
      conn.SetReadDeadline(time.Now().Add(1 * time.Second))
      // Prepare a buffer to receive the reply.
      rb := make([]byte, 1500)
      // Read from the connection to get the ICMP reply.
      n, _, err := conn.ReadFrom(rb)
      if err != nil {
        // A read error (like a timeout) means no reply was received. This is expected for hosts that are down.
        return
      }

      // Parse the received binary data back into an ICMP message.
      rm, err := icmp.ParseMessage(1, rb[:n])
      if err != nil {
        return
      }

      // Check if the message is an Echo Reply, indicating a successful ping.
      if rm.Type == ipv4.ICMPTypeEchoReply {
        // Lock the mutex to safely update the shared map.
        mu.Lock()
        ipStr := ip.String()
        // If the device is new, add it to the map
        // TODO: Is there any more info we can extract from the ICMP reply?
        if _, exists := discoveredDevices[ipStr]; !exists {
          discoveredDevices[ipStr] = &model.Device{AddrV4: ip}
        }
        discoveredDevices[ipStr].AddSource("ICMP")

        // Unlock the mutex.
        mu.Unlock()

        // To get MAC address, you'd typically look at the ARP cache AFTER a ping
        // or use platform-specific libraries. Go's net package doesn't directly expose ARP.
        // For Linux, you might read /proc/net/arp. For Windows, `arp -a`.
        // This part is complex and often requires CGO or external commands.
      }
    }(targetIP)
  }
  // Block execution until all ping goroutines have completed.
  wg.Wait()
}

// checkSSH attempts to connect to the SSH port (22) on a given host.
// It returns true if a TCP connection is successful, false otherwise.
func checkSSH(host string) bool {
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