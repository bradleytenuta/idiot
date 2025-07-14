package model

import (
	"net"
	"github.com/hashicorp/mdns"
	"github.com/rs/zerolog"
)

// Device represents a discovered device
type Device struct {
	AddrV4        net.IP                 // The IPv4 address of the device.
	AddrV6        *net.IPAddr            // The IPv6 address of the device.
	MAC           net.HardwareAddr       // The MAC address (hardware address) of the device.
	Hostname      string                 // The hostname, discovered via mDNS or reverse DNS lookup.
	Services      []*mdns.ServiceEntry   // A slice of services discovered on the device via mDNS.
	CanConnectSSH bool                   // A flag indicating whether an SSH connection can be established on port 22.
  	Sources       []string               // How the device was discovered (e.g., "mDNS", "ICMP").
}

// makes Device implement zerolog.LogObjectMarshaler for structured logging.
func (d *Device) MarshalZerologObject(e *zerolog.Event) {
	e.Str("addrV4", d.AddrV4.String())
	if d.AddrV6 != nil {
		e.Str("addrV6", d.AddrV6.String())
	}
	if d.MAC != nil {
		e.Str("mac", d.MAC.String())
	}
	if d.Hostname != "" {
		e.Str("hostname", d.Hostname)
	}
	e.Bool("canConnectSSH", d.CanConnectSSH)
	e.Strs("sources", d.Sources)

	// Custom marshaling for services to only show relevant info, like the original output.
	if len(d.Services) > 0 {
		ports := zerolog.Arr()
		for _, s := range d.Services {
			ports.Int(s.Port)
		}
		e.Array("servicePorts", ports)
	}
}

// adds a discovery source to the device's source list if it's not already present.
func (d *Device) AddSource(source string) {
	for _, s := range d.Sources {
		if s == source {
			return
		}
	}
	d.Sources = append(d.Sources, source)
}