package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ssgo",
	Short: "A static site generator",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(styleCmd)
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(deployCmd)
}
