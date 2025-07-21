package model

type Device struct {
	AddrV4        string   `yaml:"addrV4"`
	AddrV6        string   `yaml:"addrV6,omitempty"`
	MAC           string   `yaml:"mac,omitempty"`
	Hostname      string   `yaml:"hostname"`
	CanConnectSSH bool     `yaml:"canConnectSSH"`
	Sources       []string `yaml:"sources"`
}

// AddSource appends a discovery source (e.g., "ICMP", "mDNS") to the device's
// list of sources, ensuring no duplicates are added.
func (d *Device) AddSource(source string) {
	for _, s := range d.Sources {
		if s == source {
			return
		}
	}
	d.Sources = append(d.Sources, source)
}

// ListToMap converts a slice of Device structs into a map where the key is the
// device's IPv4 address. This allows for efficient lookups.
func ListToMap(devices []Device) map[string]*Device {
	deviceMap := make(map[string]*Device)
	for i := range devices {
		device := &devices[i]
		deviceMap[device.AddrV4] = device
	}
	return deviceMap
}
