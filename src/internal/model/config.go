package model

type Config struct {
	SubnetSize 		string 			`yaml:"subnet_size"`
	SelectedDevices []interface{} 	`yaml:"selected_devices,omitempty"`
	Debug 			bool 			`yaml:"debug"`
}

func NewConfig() *Config {
	return &Config{
		SubnetSize: "24",
		SelectedDevices: []interface{}{},
		Debug: false,
	}
}