package model

type Device struct {
	AddrV4        string   	`yaml:"addrV4"`
	AddrV6        string   	`yaml:"addrV6,omitempty"`
	MAC           string   	`yaml:"mac,omitempty"`
	Hostname      string   	`yaml:"hostname"`
	CanConnectSSH bool		`yaml:"canConnectSSH"`
  	Sources       []string 	`yaml:"sources"`
}

func (d *Device) AddSource(source string) {
	for _, s := range d.Sources {
		if s == source {
			return
		}
	}
	d.Sources = append(d.Sources, source)
}