package cmd

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"com.bradleytenuta/idiot/internal"
)

var rootCmd = &cobra.Command{
	Use:   "idiot",
	Short: "Enables you to identify and manage internet of things (IOT).",
	Long:  `A GO command line interface, that enables you to identify and manage internet of things (IOT) on your local network.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// init is a special Go function that is called when the package is initialized.
// It registers the initConfig function to be called by Cobra when it initializes.
func init() {
	cobra.OnInitialize(initConfig)
	// Disable the default 'completion' command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in a config file and ENV variables if set. It also sets up
// the global logger based on the debug configuration. If a configuration file
// does not exist, it creates a default one.
func initConfig() {
	executablePath, _ := os.Executable()
	configFilePath := filepath.Join(filepath.Dir(executablePath), "configuration.yaml")
	exists, _ := internal.FileExists(configFilePath)

	if !exists {
		err := internal.WriteConfigFile(configFilePath)
		if err != nil {
			log.Error().Msgf("Error writing new configuration file: %v", err)
			return
		}
	}

	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Error().Msgf("Error using configuration file: %v", err)
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if viper.GetBool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logs are turned on!")
	}
}
