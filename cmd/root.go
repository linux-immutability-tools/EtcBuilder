package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fsguard",
	Short: "A tool for verifying filesystem integrity",
}

func init() {
	rootCmd.AddCommand(NewBuildCommand())
}

func Execute() error {
	return rootCmd.Execute()
}
