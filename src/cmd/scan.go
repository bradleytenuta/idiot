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
)

// Device represents a discovered device
type Device struct {
  AddrV4    net.IP
  AddrV6    *net.IPAddr
  MAC       net.HardwareAddr
  Hostname  string // From mDNS or reverse DNS
  Services  []*mdns.ServiceEntry // Services discovered via mDNS
  IsReachable bool // From ping
}

func init() {
  rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
  Use:   "scan",
  Short: "Scan the local network of this device and list the IP Addresses of devices connected to it.",
  Long:  `Scan the local network of this device and list the IP Addresses of devices connected to it. Including IPv4, IPv6 and ports reachable.`,
  Run: func(cmd *cobra.Command, args []string) {
    ifaces, err := net.Interfaces()
    if err != nil {
      fmt.Printf("Error getting interfaces: %v\n", err)
      return
    }

    var localIP net.IP
    var subnetMask net.IPMask
    var iface *net.Interface // Pointer to the interface found

    for _, i := range ifaces {
      // TODO: replace with viper.Get("network_name")
      if strings.Contains(i.Name, "Ethernet 2") { // Or whatever interface name you need
        addrs, err := i.Addrs()
        if err != nil {
          fmt.Printf("Error getting addresses for %s: %v\n", i.Name, err)
          continue
        }
        for _, a := range addrs {
          if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
            if ipNet.IP.To4() != nil { // We only care about IPv4 for this example
              localIP = ipNet.IP.To4() // Even if IPv4 (length == 4), Go will store this in an IPv6 format with length == 16.
              subnetMask = ipNet.Mask
              iface = &i // Store the pointer to the interface
              fmt.Printf("Found Local IP: %s/%s on interface: %s\n", localIP.String(), net.IP(subnetMask).String(), i.Name)
              break
            }
          }
        }
      }
      if localIP != nil {
        break
      }
    }

    if localIP == nil {
      fmt.Println("Could not find a suitable IPv4 address on 'Ethernet 2'. Exiting.")
      return
    }

    // Calculate network address and broadcast address
    networkAddr := localIP.Mask(subnetMask)
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

    discoveredDevices := make(map[string]*Device) // Use string representation of IP as key
    var mu sync.Mutex // Mutex to protect discoveredDevices map

    // 1. mDNS Discovery (Service Discovery)
    // This will populate devices that advertise services.
    mdnsEntries := make(chan *mdns.ServiceEntry, 100) // Buffer the channel
    go func() {
      defer close(mdnsEntries)
      params := mdns.DefaultParams("_services._dns-sd._udp") // Discover all advertised services
      params.Timeout = 5 * time.Second
      params.Entries = mdnsEntries
      params.DisableIPv6 = true
      if iface != nil {
        params.Interface = iface
      }
      fmt.Println("Starting mDNS discovery...")
      err := mdns.Query(params)
      if err != nil {
        fmt.Printf("mDNS query error: %v\n", err)
      }
    }()

    go func() {
      for entry := range mdnsEntries {
        mu.Lock()
        ipStr := entry.Addr.String()
        if _, exists := discoveredDevices[ipStr]; !exists {
          discoveredDevices[ipStr] = &Device{AddrV4: entry.AddrV4, AddrV6: entry.AddrV6IPAddr, Hostname: entry.Host}
        }
        discoveredDevices[ipStr].Services = append(discoveredDevices[ipStr].Services, entry)
        mu.Unlock()
        fmt.Printf("mDNS discovered: IP=%s, Host=%s\n", entry.Addr, entry.Host)
      }
    }()

    // 2. ARP/ICMP Scan (Host Discovery)
    // Iterate through all possible IPs in the subnet and try to ping them.
    fmt.Println("Starting ICMP/ARP scan...")
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

    for i := ipRangeStart + 1; i < ipRangeEnd; i++ { // Skip network and broadcast addresses
      targetIP := make(net.IP, 4)
      copy(targetIP, networkAddr)
      targetIP[3] = byte(i) // Only works for /24 and smaller ranges within the last octet

      wg.Add(1)
      go func(ip net.IP) {
        defer wg.Done()
        conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
        if err != nil {
          //fmt.Printf("Error listening for ICMP: %v\n", err)
          return
        }
        defer conn.Close()

        wm := icmp.Message{
          Type: ipv4.ICMPTypeEcho, Code: 0,
          Body: &icmp.Echo{
            ID: os.Getpid() & 0xffff, Seq: 1,
            Data: []byte("HELLO-R-U-THERE"),
          },
        }
        wb, err := wm.Marshal(nil)
        if err != nil {
          return
        }

        _, err = conn.WriteTo(wb, &net.IPAddr{IP: ip})
        if err != nil {
          fmt.Printf("Error sending ICMP to %s: %v\n", ip.String(), err)
          return
        }

        conn.SetReadDeadline(time.Now().Add(1 * time.Second))
        rb := make([]byte, 1500)
        n, peer, err := conn.ReadFrom(rb)
        if err != nil {
          fmt.Printf("No ICMP response from %s: %v\n", ip.String(), err)
          return
        }

        rm, err := icmp.ParseMessage(1, rb[:n])
        if err != nil {
          return
        }

        if rm.Type == ipv4.ICMPTypeEchoReply {
          mu.Lock()
          ipStr := ip.String()
          if _, exists := discoveredDevices[ipStr]; !exists {
            discoveredDevices[ipStr] = &Device{AddrV4: ip, IsReachable: true}
          } else {
            discoveredDevices[ipStr].IsReachable = true
          }
          mu.Unlock()
          fmt.Printf("Ping response from: %s\n", peer.String())

          // To get MAC address, you'd typically look at the ARP cache AFTER a ping
          // or use platform-specific libraries. Go's net package doesn't directly expose ARP.
          // For Linux, you might read /proc/net/arp. For Windows, `arp -a`.
          // This part is complex and often requires CGO or external commands.
        }
      }(targetIP)
    }
    wg.Wait() // Wait for all pings to complete

    // Wait a bit longer to allow mDNS to complete, or use a context with timeout
    time.Sleep(7 * time.Second)

    fmt.Println("\n--- Discovered Devices ---")
    for _, dev := range discoveredDevices {
      fmt.Printf("AddrV4: %s, AddrV6: %s", dev.AddrV4, dev.AddrV6)
      if dev.Hostname != "" {
        fmt.Printf(", Hostname: %s", dev.Hostname)
      }
      if dev.IsReachable {
        fmt.Printf(", Reachable: Yes")
      }
      if len(dev.Services) > 0 {
        fmt.Printf(", Services:")
        for _, s := range dev.Services {
          fmt.Printf(" [(%v)]", s.Port)
        }
      }
      fmt.Println()
    }
  },
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