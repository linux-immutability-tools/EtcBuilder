package cmd

import (
	"fmt"

	"github.com/linux-immutability-tools/EtcBuilder/core"
	"github.com/spf13/cobra"
)

type UnknownFileError struct {
	file string
}

func (m *UnknownFileError) Error() string {
	return fmt.Sprintf("File type of %s is not supported", m.file)
}

type FileHandler struct {
	IsFileSupported func(path string) bool
	Handle          func(relativeFilePath, oldSysDir, newSysDir, oldUserDir, newUserDir string) error
}

var ExternalFileHandlers []FileHandler

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
	} else if len(args) <= 3 {
		return fmt.Errorf("not enough directories specified")
	}

	oldSys := args[0]
	newSys := args[1]
	oldUser := args[2]
	newUser := args[3]

	return ExtBuildCommand(oldSys, newSys, oldUser, newUser)
}

func ExtBuildCommand(oldSys, newSys, oldUser, newUser string) error {

	err := core.BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		return err
	}

	return nil
}
