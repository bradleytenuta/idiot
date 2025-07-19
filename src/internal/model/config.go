package model

type Config struct {
	SelectedDevices []interface{} `yaml:"selected_devices,omitempty"`
	Debug           bool          `yaml:"debug"`
}

func NewConfig() *Config {
	return &Config{
		SelectedDevices: []interface{}{},
		Debug:           false,
	}
}
