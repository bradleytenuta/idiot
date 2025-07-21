package cmd

import (
	"github.com/spf13/cobra"
)

// init registers the version command with the root command.
func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Idiot.",
	Long:  `All software has versions. This is Idiots's.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("1.0.0")
	},
}
