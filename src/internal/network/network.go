package network

import (
	"fmt"
	"net"
)

// Gets the preferred outbound ip of this machine.
// It works by dialing a connection to a public DNS server (without sending any data)
// and then checking the local address of the connection.
func getOutboundIP() (net.IP, error) {
	// The address doesn't have to be reachable, it's just used to find the route.
	// Using a public DNS server is a common and reliable choice.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("could not dial to determine outbound IP: %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("could not assert local address to UDPAddr")
	}

	return localAddr.IP, nil
}

// GetInternetFacingNetworkInfo automatically discovers the network interface used for
// internet connectivity and returns its network information.
func GetInternetFacingNetworkInfo() (net.IP, net.IP, *net.Interface, error) {
	// First, determine the local IP address the OS uses for outbound traffic.
	outboundIP, err := getOutboundIP()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get outbound IP: %w", err)
	}

	// Now, find the interface that owns this IP address.
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting interfaces: %w", err)
	}

	var subnetMask net.IPMask
	var selectedIface *net.Interface // Pointer to the interface found

	// Iterate over all found network interfaces by index to safely get a pointer.
	for i := range ifaces {
		// Use a local variable for the current interface for readability.
		currentIface := ifaces[i]
		// Get all addresses associated with the current interface.
		addrs, err := currentIface.Addrs()
		if err != nil {
			// Log the error but continue, as there might be other matching interfaces.
			fmt.Printf("Warning: could not get addresses for %s: %v\n", currentIface.Name, err)
			continue
		}

		// Iterate over the addresses of the interface.
		for _, addr := range addrs {
			var ip net.IP
			// Check if the address is an IP network and not a loopback address.
			if ipNet, ok := addr.(*net.IPNet); ok {
				ip = ipNet.IP
			}

			// Check if the IP from the interface matches our outbound IP.
			if ip != nil && ip.Equal(outboundIP) {
				// We are interested in IPv4 addresses for this scan.
				if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
					// Store the IPv4 address, subnet mask, and a pointer to the interface.
					// Even if IPv4 (length == 4), Go will store this in an IPv6 format with length == 16.
					// This the local IP address of the current device.
					subnetMask = ipNet.Mask
					selectedIface = &ifaces[i] // Safely get the address of the slice element.
					break // Exit the address loop once a suitable IPv4 address is found.
				}
			}
		}
		if selectedIface != nil {
			break
		}
	}

	if selectedIface == nil {
		return nil, nil, nil, fmt.Errorf("could not find an interface for outbound IP %s", outboundIP.String())
	}

	fmt.Printf("Found Local IP: %s/%s on interface: %s\n", outboundIP.String(), net.IP(subnetMask).String(), selectedIface.Name)

	// --- Subnet Calculation ---
  	// Calculate the network address by applying the subnet mask to the local IP.
	networkAddr := outboundIP.Mask(subnetMask)
	// Prepare a slice to hold the broadcast address.
	broadcastAddr := make(net.IP, len(outboundIP))
	for i := 0; i < len(outboundIP); i++ {
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