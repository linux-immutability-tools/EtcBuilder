package cmd

import (
	"errors"
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

type NoMatchError struct{}

func (m *NoMatchError) Error() string {
	return "Specified files are not the same"
}

type NotAFileError struct{}

func (m *NotAFileError) Error() string {
	return "Specified paths do not point to regular file"
}

func NewBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "build",
		Short:        "Build a etc overlay based on the given System and User etc",
		RunE:         buildCommand,
		SilenceUsage: true,
	}

	return cmd
}

func getCommonPath(s1 string, s2 string) string {
	paths1 := strings.Split(s1, "/")
	paths2 := strings.Split(s2, "/")
	var commonPath string
	if strings.HasPrefix(s1, "/") {
		commonPath = "/"
	}

	for i, path := range paths1 {
		if filepath.Join(commonPath, path) == filepath.Join(commonPath, paths2[i]) {
			commonPath = filepath.Join(commonPath, path)
		} else {
			break
		}
	}
	return commonPath
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

func fileHandler(userFile string, newSysFile string, fileInfo fs.FileInfo, newFileInfo fs.FileInfo, newSys string, oldSys string, newUser string, oldUser string) error {
	if fileInfo.IsDir() || newFileInfo.IsDir() {
		return &NotAFileError{}
	} else if strings.HasPrefix(fileInfo.Mode().Type().String(), "L") {
		sym, err := filepath.EvalSymlinks(userFile)
		if strings.HasSuffix(err.Error(), "no such file or directory") {
			return nil
		} else if err != nil {
			return err
		}
		linkPath := strings.TrimPrefix(sym, getCommonPath(userFile, sym)+"/")
		err = os.Chdir(newUser)
		if err != nil {
			return err
		}
		err = os.Symlink(strings.TrimPrefix(linkPath, getCommonPath(userFile, sym)), strings.TrimPrefix(userFile, getCommonPath(userFile, sym)+"/"))
		return err
	} else if strings.ReplaceAll(userFile, oldUser, "") != strings.ReplaceAll(newSysFile, newSys, "") {
		return &NoMatchError{}
	}

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
			dirInfo, err := os.Stat(strings.TrimRight(userFile, fileInfo.Name()))
			if err != nil {
				return err
			}
			destFilePath := newUser + "/" + strings.ReplaceAll(userFile, oldUser, "")
			os.MkdirAll(strings.TrimRight(destFilePath, fileInfo.Name()), dirInfo.Mode())
			err = copyFile(userFile, destFilePath)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

	return ExtBuildCommand(oldSys, newSys, oldUser, newUser)
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

	var userPaths []string
	err = filepath.Walk(oldUser, func(userPath string, userInfo os.FileInfo, e error) error {
		isInSys := false
		if userInfo.IsDir() {
			return nil
		}
		err := filepath.Walk(newSys, func(newPath string, newInfo os.FileInfo, err error) error {
			err = fileHandler(userPath, newPath, userInfo, newInfo, newSys, oldSys, newUser, oldUser)
			if err == nil {
				isInSys = true
			} else if errors.Is(err, &NoMatchError{}) || errors.Is(err, &NotAFileError{}) {
				return nil
			}
			return err
		})
		if isInSys == false {
			userPaths = append(userPaths, userPath)
		}
		return err
	})
	if err != nil {
		return err
	}
	for _, userFile := range userPaths {
		fmt.Printf("Copying user file %s\n", userFile)
		fileInfo, err := os.Stat(userFile)
		dirInfo, err := os.Stat(strings.TrimRight(userFile, fileInfo.Name()))
		if err != nil {
			return err
		}
		destFilePath := newUser + "/" + strings.ReplaceAll(userFile, oldUser, "")
		os.MkdirAll(strings.TrimRight(destFilePath, fileInfo.Name()), dirInfo.Mode())
		copyFile(userFile, destFilePath)
	}
	return nil
}
