package network

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"com.bradleytenuta/idiot/internal/model"
)

// PerformIcmpScan discovers hosts on the local network by sending ICMP echo requests.
// It uses a single listener and concurrent routines for sending pings and reading replies,
// which is significantly more performant than creating a listener for each ping.
func PerformIcmpScan(networkAddr, broadcastAddr net.IP, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	// Listen for ICMP packets on all available IPv4 interfaces.
	// We create one listener for the entire scan duration for efficiency.
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Error().Msgf("Failed to listen for ICMP packets: %v", err)
		return
	}
	defer conn.Close()

	// Use a context to manage the scan's lifecycle, ensuring it stops after a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Start a dedicated goroutine to read all incoming ICMP replies.
	wg.Add(1)
	go readReplies(ctx, conn, discoveredDevices, mu, &wg)

	// Start a dedicated goroutine to send out all the pings.
	wg.Add(1)
	go sendPings(conn, networkAddr, broadcastAddr, &wg)

	// Wait for both the sender and reader goroutines to complete.
	wg.Wait()
}

// readReplies runs in a dedicated goroutine, listening for ICMP echo replies
// until the context is cancelled.
func readReplies(ctx context.Context, conn *icmp.PacketConn, discoveredDevices map[string]*model.Device, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	replyBuf := make([]byte, 1500)

	for {
		select {
		case <-ctx.Done(): // The scan timeout has been reached.
			return
		default:
			// Set a short deadline to make the ReadFrom call non-blocking,
			// allowing the loop to check the context cancellation status.
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := conn.ReadFrom(replyBuf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Expected timeout, continue loop to check context.
				}
				log.Debug().Msgf("ICMP read error: %v", err)
				return // Other errors are fatal for the reader.
			}

			// Parse the reply and update the device map if it's a valid echo reply.
			msg, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), replyBuf[:n])
			if err != nil {
				continue
			}

			if msg.Type == ipv4.ICMPTypeEchoReply {
				if ipAddr, ok := addr.(*net.IPAddr); ok {
					updateDiscoveredDevice(ipAddr.IP, discoveredDevices, mu)
				}
			}
		}
	}
}

// sendPings generates all IPs in the subnet and sends an ICMP echo request to each.
func sendPings(conn *icmp.PacketConn, networkAddr, broadcastAddr net.IP, wg *sync.WaitGroup) {
	defer wg.Done()

	// Construct the ICMP Echo Request message once.
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff, // Use process ID to uniquely identify this pinger.
			Seq:  1,
			Data: []byte("IDIOT-SCAN"),
		},
	}
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		log.Error().Msgf("Failed to marshal ICMP message: %v", err)
		return
	}

	// Iterate through all valid host IPs in the subnet and send a ping.
	for _, ip := range generateIPs(networkAddr, broadcastAddr) {
		conn.WriteTo(msgBytes, &net.IPAddr{IP: ip})
		time.Sleep(1 * time.Millisecond) // Small delay to avoid flooding the network.
	}
}

// updateDiscoveredDevice safely adds or updates a device in the shared map.
func updateDiscoveredDevice(ip net.IP, discoveredDevices map[string]*model.Device, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	ipStr := ip.String()
	if _, exists := discoveredDevices[ipStr]; !exists {
		discoveredDevices[ipStr] = &model.Device{AddrV4: ipStr}
	}
	discoveredDevices[ipStr].AddSource("ICMP")
}

// generateIPs creates a slice of all valid host IP addresses within a given
// network range, excluding the network and broadcast addresses.
// This is done by converting IPs to integers for robust iteration.
func generateIPs(networkAddr, broadcastAddr net.IP) []net.IP {
	// Ensure we are working with 4-byte IPv4 addresses.
	network := networkAddr.To4()
	broadcast := broadcastAddr.To4()
	if network == nil || broadcast == nil {
		return nil
	}

	// Convert IPs to uint32 for easy iteration.
	start := binary.BigEndian.Uint32(network)
	end := binary.BigEndian.Uint32(broadcast)

	var ips []net.IP
	// Iterate from the first host IP to the last host IP.
	for i := start + 1; i < end; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		ips = append(ips, ip)
	}
	return ips
}
