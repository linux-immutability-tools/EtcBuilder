//go:build !nodevfiles

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func init() {
	ExternalFileHandlers = append(ExternalFileHandlers, FileHandler{checkIfDeviceFile, copyDeviceFile})
}

func checkIfDeviceFile(filePath string) bool {
	info, err := os.Lstat(filePath)
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.Mode().Type().String(), "D") {
		return true
	}
	return false
}

func copyDeviceFile(relFromPath, oldSysDir, newSysDir, oldUserDir, newUserDir string) error {
	fmt.Println("Keeping user device file", relFromPath)

	fromPath := filepath.Join(oldUserDir, relFromPath)
	toPath := filepath.Join(newUserDir, relFromPath)

	var stat syscall.Stat_t
	err := syscall.Lstat(fromPath, &stat)
	if err != nil {
		return err
	}
	err = syscall.Mknod(toPath, stat.Mode, int(stat.Rdev))
	return err
}
