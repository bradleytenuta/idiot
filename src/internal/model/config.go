package model

type Config struct {
	SelectedDevices []interface{} `yaml:"selected_devices,omitempty"`
	Debug           bool          `yaml:"debug"`
	SshSecureMode   bool          `yaml:"ssh_secure_mode"`
}

// NewConfig creates and returns a new Config struct with default values.
func NewConfig() *Config {
	return &Config{
		SelectedDevices: []interface{}{},
		Debug:           false,
		SshSecureMode:   true,
	}
}
