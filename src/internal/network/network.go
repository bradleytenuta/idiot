package network

import (
	"errors"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

// GetInternetFacingNetworkInfo automatically discovers the network interface used for
// internet connectivity and returns its network information.
func GetInternetFacingNetworkInfo() (net.IP, net.IP, *net.Interface, error) {
	// First, determine the local IP address the OS uses for outbound traffic.
	outboundIP, err := getOutboundIP()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get outbound IP: %w", err)
	}

	// Find the interface and IP network configuration for our outbound IP.
	selectedIface, ipNet, err := findInterfaceForIP(outboundIP)
	if err != nil {
		log.Debug().Msgf("could not find interface for outbound IP %s: %v", outboundIP, err)
		return nil, nil, nil, fmt.Errorf("could not find interface for outbound IP %s: %w", outboundIP, err)
	}

	subnetMask := ipNet.Mask
	log.Debug().Msgf("Found Local IP: %s/%s on interface: %s", outboundIP.String(), net.IP(subnetMask).String(), selectedIface.Name)

	// Calculate the network address by applying the subnet mask to the local IP.
	networkAddr := outboundIP.Mask(subnetMask)
	// Calculate broadcast by ORing the network address with the inverted subnet mask.
	broadcastAddr := make(net.IP, len(outboundIP))
	for i := 0; i < len(outboundIP); i++ {
		broadcastAddr[i] = networkAddr[i] | ^subnetMask[i]
	}
	log.Debug().Msgf("Network Address: %s", networkAddr.String())
	log.Debug().Msgf("Broadcast Address: %s", broadcastAddr.String())

	return networkAddr, broadcastAddr, selectedIface, nil
}

// Gets the preferred outbound ip of this machine.
// It works by dialing a connection to a public DNS server (without sending any data)
// and then checking the local address of the connection.
func getOutboundIP() (net.IP, error) {
	// The address doesn't have to be reachable, it's just used to find the route.
	// Using a public DNS server is a common and reliable choice.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Debug().Msgf("could not dial to determine outbound IP: %v", err)
		return nil, err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		log.Debug().Msg("could not assert local address to UDPAddr")
		return nil, errors.New("could not assert local address to *net.UDPAddr")
	}

	return localAddr.IP, nil
}

// findInterfaceForIP iterates through system network interfaces to find the one
// associated with the given IP address, returning the interface and its IP network configuration.
func findInterfaceForIP(ip net.IP) (*net.Interface, *net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get network interfaces: %w", err)
	}

	for i := range ifaces {
		iface := &ifaces[i]
		addrs, err := iface.Addrs()
		if err != nil {
			log.Debug().Msgf("Cannot get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			// Check if the address is an IPNet and if its IP matches the one we're looking for.
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.Equal(ip) {
				// We found the correct interface and IP network.
				return iface, ipNet, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no interface found for IP %s", ip)
}
