package cmd

import (
	"os"
	"log"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	configFilePath     string
	userLicense        string

	rootCmd = &cobra.Command{
		Use:   "cobra-cli",
		Short: "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // File or directory exists
	}
	if os.IsNotExist(err) {
		return false, nil // File or directory does not exist
	}
	return false, err // Other error (e.g., permissions)
}

func initConfig() {
	executablePath, err := os.Executable()
	configFilePath = filepath.Join(filepath.Dir(executablePath), "configuration.yaml")
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}

	// Read config file if present:
	exists, err := fileExists(configFilePath)
	if (err != nil) {
		log.Fatalf("Error while reading configuration file: %v", err)
	}
	if (!exists) {
		// Otherwise create config file with default values.
		type Config struct {
			NetworkName string `yaml:"network_name"`
			SubnetSize string `yaml:"subnet_size"`
		}
		// TODO: maybe update to read in yaml file instead.
		// TODO: Update all code to use log instead of fmt.
		defaultConfig := Config{
			NetworkName: "<network name here>",
			SubnetSize: "24",
		}
		yamlBytes, err := yaml.Marshal(defaultConfig)
		if err != nil {
			log.Fatalf("Error marshalling YAML: %v", err)
		}
		err = os.WriteFile(configFilePath, yamlBytes, 0644)
		if err != nil {
			log.Fatalf("Error writing YAML to file %s: %v", configFilePath, err)
		}
		log.Printf("Successfully wrote YAML content to '%s'\n", configFilePath)
	}

	// store config file
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err == nil {
		// TODO replace log and print with zerolog.
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Fatalf("Error when finding config file: %v", err)
	}
}
