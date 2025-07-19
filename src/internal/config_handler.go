package internal

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

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

func SaveSelectedIotDevice(iotDevice *model.Device) {
	// Retrieve the current list of devices from the configuration.
	var configDevices []model.Device
	if err := viper.UnmarshalKey("selected_devices", &configDevices); err != nil {
		log.Error().Msgf("Failed to read 'selected_devices' from config: %v", err)
		return
	}

	// Check for duplicates using the string representation of the IP address.
	isDuplicate := false
	for _, cd := range configDevices {
		if cd.AddrV4 == iotDevice.AddrV4 {
			isDuplicate = true
			break
		}
	}

	if isDuplicate {
		log.Debug().Msgf("Device '%s' is already in the list. No changes made.", iotDevice.AddrV4)
	} else {
		// Append the new, serializable device to the list.
		configDevices = append(configDevices, *iotDevice)

		// Set the updated slice back into viper.
		viper.Set("selected_devices", configDevices)

		// Write the changes to the configuration file.
		if err := viper.WriteConfig(); err != nil {
			log.Error().Msgf("Error writing configuration file: %v", err)
		}
		log.Debug().Msgf("Successfully added '%s' to 'selected_devices' in the configuration file.", iotDevice.AddrV4)
	}
}
