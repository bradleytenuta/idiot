package cmd
// TODO: Reformat all files. Ensure all have 2 spaces.
import (
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
	"com.bradleytenuta/idiot/internal"
)

var (
	rootCmd = &cobra.Command{
		Use:   "idiot",
		Short: "Enables you to identify and manage internet of things (IOT).",
		Long: `A GO command line interface, that enables you to identify and manage internet of things (IOT) on your local network.`,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	executablePath, _ := os.Executable()
	configFilePath := filepath.Join(filepath.Dir(executablePath), "configuration.yaml")
	exists, _ := internal.FileExists(configFilePath)

	if (!exists) {
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