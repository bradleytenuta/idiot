package network

import (
  "net"
  "time"
  "sync"
  "os"
  "golang.org/x/net/icmp"
  "golang.org/x/net/ipv4"
  "com.bradleytenuta/idiot/internal/model"
)

// Scans the local network using ICMP echo requests (pings) to discover hosts.
// It iterates through the IP range defined by the network and broadcast addresses.
// For each responsive host, it adds or updates an entry in the shared discoveredDevices map.
func PerformIcmpScan(networkAddr, broadcastAddr net.IP, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
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