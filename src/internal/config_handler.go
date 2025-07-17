package internal

import (
	"os"
	"gopkg.in/yaml.v3"
	"github.com/spf13/viper"
  "github.com/rs/zerolog/log"
	"com.bradleytenuta/idiot/internal/model"
)

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Other error (e.g., permissions)
	return false, err
}

func WriteConfigFile(configFilePath string) error {
	defaultConfig := model.NewConfig()
	yamlBytes, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}
	// The number 0644 represents a standard file permission mode used in Linux & macOS
	return os.WriteFile(configFilePath, yamlBytes, 0644)
}

func ReadIotDevices() []model.Device {
	var iotDevices []model.Device
	if err := viper.UnmarshalKey("selected_devices", &iotDevices); err != nil {
    log.Error().Msgf("Error reading IOT Devices from the configuration file: %v", err)
		return []model.Device{}
	}
	return iotDevices
}