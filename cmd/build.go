package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/linux-immutability-tools/EtcBuilder/core"
	"github.com/linux-immutability-tools/EtcBuilder/settings"
	"golang.org/x/exp/slices"

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

func copyFile(source string, target string) error {

	fin, err := os.Open(source)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(target)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)

	if err != nil {
		return err
	}

	return nil
}

func clearDirectory(fileList []fs.DirEntry, root string) error {
	for _, file := range fileList {
		if file.IsDir() {
			files, err := os.ReadDir(root + "/" + file.Name())
			if err != nil {
				return err
			}

			err = clearDirectory(files, root+"/"+file.Name())
			if err != nil {
				return err
			}
		}
		err := os.Remove(root + "/" + file.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFileWithDirs(oldPath string, oldBasePath string, newBasePath string, filename string) error {
	dirInfo, err := os.Stat(strings.TrimRight(oldPath, filename))
	if err != nil {
		return err
	}
	destFilePath := newBasePath + "/" + strings.ReplaceAll(oldPath, oldBasePath, "")
	err = os.MkdirAll(strings.TrimRight(destFilePath, filename), dirInfo.Mode())
	if err != nil {
		return err
	}
	err = copyFile(oldPath, destFilePath)
	return err
}

func fileHandler(userFile string, newSysFile string, fileInfo fs.FileInfo, newFileInfo fs.FileInfo, newSys string, oldSys string, newUser string, oldUser string) error {
	if slices.Contains(settings.SpecialFiles, strings.ReplaceAll(newSysFile, strings.TrimRight(newSys, "/")+"/", "")) {
		fmt.Printf("Special merging file %s\n", fileInfo.Name())
		err := core.MergeSpecialFile(userFile, oldSys+"/"+strings.ReplaceAll(userFile, oldUser, ""), newSysFile, newUser+"/"+strings.ReplaceAll(userFile, oldUser, ""))
		if err != nil {
			return err
		}
	} else if slices.Contains(settings.OverwriteFiles, fileInfo.Name()) {
		fmt.Printf("Overwriting User %[1]s with New %[1]s!\n", fileInfo.Name()) // Don't have to do anything when overwriting
	} else {
		keep, err := core.KeepUserFile(userFile, newSysFile)
		if err != nil {
			return err
		}
		if keep {
			fmt.Printf("Keeping User file %s\n", userFile)
			err = copyFileWithDirs(userFile, oldUser, newUser, fileInfo.Name())
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func buildNewSys(oldSys string, newSys string, oldUser string, newUser string) error {
	err := filepath.Walk(oldUser, func(userPath string, userInfo os.FileInfo, e error) error {
		if userInfo.IsDir() {
			return nil
		}
		fileInNewSys := false
		err := filepath.Walk(newSys, func(newPath string, newInfo os.FileInfo, err error) error {
			if newInfo.IsDir() {
				return nil
			}
			if strings.ReplaceAll(userPath, oldUser, "") != strings.ReplaceAll(newPath, newSys, "") {
				return nil
			}

			fileInNewSys = true
			return fileHandler(userPath, newPath, userInfo, newInfo, newSys, oldSys, newUser, oldUser)
		})
		if err != nil {
			return err
		}
		if !fileInNewSys {
			fmt.Printf("Keeping User file %s\n", userPath)
			err = copyFileWithDirs(userPath, oldUser, newUser, userInfo.Name())
			if err != nil {
				return err
			}
		}
		return err
	})
	return err
}

func buildCommand(_ *cobra.Command, args []string) error {
	if len(args) <= 0 {
		return fmt.Errorf("no etc directories specified")
	} else if len(args) <= 3 {
		return fmt.Errorf("not enough directories specified")
	}

	err := settings.GatherConfigFiles()
	if err != nil {
		return err
	}

	oldSys := args[0]
	newSys := args[1]
	oldUser := args[2]
	newUser := args[3]

	newUserFiles, err := os.ReadDir(newUser)
	if err != nil {
		return err
	}

	err = clearDirectory(newUserFiles, newUser)
	if err != nil {
		return err
	}

	err = buildNewSys(oldSys, newSys, oldUser, newUser)

	return err
}

func ExtBuildCommand(oldSys string, newSys string, oldUser string, newUser string) error {
	err := settings.GatherConfigFiles()
	if err != nil {
		return err
	}

	destFiles, err := os.ReadDir(newUser)
	if err != nil {
		return err
	}

	err = clearDirectory(destFiles, newUser)
	if err != nil {
		return err
	}

	err = buildNewSys(oldSys, newSys, oldUser, newUser)

	return err
}
