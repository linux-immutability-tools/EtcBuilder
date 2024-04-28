package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/linux-immutability-tools/EtcBuilder/core"
	"github.com/linux-immutability-tools/EtcBuilder/settings"
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
	err := settings.GatherConfigFiles()
	if err != nil {
		return err
	}

	err = clearDirectory(newUser)
	if err != nil {
		return err
	}

	err = filepath.Walk(oldUser, func(userPath string, userInfo os.FileInfo, e error) error {
		if userInfo.IsDir() {
			return nil
		}
		userPathRel := strings.TrimPrefix(userPath, oldUser)
		userPathRel = strings.TrimPrefix(userPathRel, "/")

		err = copyUserFile(userPathRel, oldSys, newSys, oldUser, newUser, userInfo)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func copyUserFile(relUserFile, oldSysDir, newSysDir, oldUserDir, newUserDir string, oldFileInfo os.FileInfo) error {
	userFileType := oldFileInfo.Mode().Type().String()

	absUserFileOld := filepath.Join(oldUserDir, relUserFile)
	absUserFileNew := filepath.Join(newUserDir, relUserFile)

	os.MkdirAll(filepath.Dir(absUserFileNew), 0o755)

	for _, externalFileHandler := range ExternalFileHandlers {
		if !externalFileHandler.IsFileSupported(absUserFileOld) {
			continue
		}
		return externalFileHandler.Handle(relUserFile, oldSysDir, newSysDir, oldUserDir, newUserDir)
	}

	if slices.Contains(settings.SpecialFiles, relUserFile) {
		fmt.Println("Special merging file", relUserFile)
		absSysFileOld := filepath.Join(oldSysDir, relUserFile)
		absSysFileNew := filepath.Join(newSysDir, relUserFile)
		absUserFileNew := filepath.Join(newUserDir, relUserFile)
		err := core.MergeSpecialFile(absUserFileOld, absSysFileOld, absSysFileNew, absUserFileNew)
		if err != nil {
			return err
		}
		return nil
	} else if oldFileInfo.Mode().IsRegular() {
		fmt.Println("Keeping user file", relUserFile)
		err := copyRegular(absUserFileOld, absUserFileNew)
		return err
	} else if strings.HasPrefix(userFileType, "L") {
		fmt.Println("Keeping user symlink", relUserFile)
		err := copySymlink(absUserFileOld, absUserFileNew)
		return err
	} else {
		return &UnknownFileError{absUserFileOld}
	}
}

func clearDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			err = clearDirectory(entryPath)
			if err != nil {
				return err
			}
		}
		err := os.Remove(entryPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyRegular(fromPath, toPath string) error {
	source, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func copySymlink(fromPath, toPath string) error {
	sym, err := os.Readlink(fromPath)
	if err != nil {
		return err
	}
	err = os.Symlink(sym, toPath)
	return err
}
