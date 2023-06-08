package cmd

import (
	"fmt"
	"github.com/linux-immutability-tools/EtcBuilder/core"
	"github.com/spf13/cobra"
)

func NewBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "build",
		Short:        "Build a etc overlay based on the given System and User etc",
		RunE:         buildCommand,
		SilenceUsage: true,
	}

	return cmd
}

func buildCommand(_ *cobra.Command, args []string) error {
	if len(args) <= 0 {
		return fmt.Errorf("no etc directories specified")
	} else if len(args) <= 2 {
		return fmt.Errorf("not enough directories specified")
	}
	core.MergeSpecialFile("./passwd.user", "./passwd.old", "./passwd.new")
	return nil
}
