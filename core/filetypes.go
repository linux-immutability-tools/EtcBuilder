package core

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"syscall"
	"time"
)

type Symlink struct{}

func (s *Symlink) SupportsFile(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

func (s *Symlink) Copy(fromInfo os.FileInfo, from, to string) error {
	target, err := os.Readlink(from)
	if err != nil {
		return fmt.Errorf("can't read symlink: %w", err)
	}

	err = os.Symlink(target, to)
	if err != nil {
		return fmt.Errorf("can't create symlink: %w", err)
	}

	return nil
}

func (s *Symlink) CopyAttributes(fromInfo os.FileInfo, to string) error {
	return nil
}

func (s *Symlink) IsIdentical(a, b os.FileInfo, aPath, bPath string) (bool, error) {
	aTarget, err := os.Readlink(aPath)
	if err != nil {
		return false, err
	}

	bTarget, err := os.Readlink(bPath)
	if err != nil {
		return false, err
	}

	return aTarget == bTarget, nil
}

type Folder struct{}

func (f *Folder) SupportsFile(info os.FileInfo) bool {
	return info.Mode().IsDir()
}

func (f *Folder) Copy(fromInfo os.FileInfo, from, to string) error {
	err := os.Mkdir(to, fromInfo.Mode())
	if err != nil {
		return fmt.Errorf("can't make directory: %w", err)
	}

	return nil
}

func (f *Folder) CopyAttributes(fromInfo os.FileInfo, to string) error {
	return copyAttributes(fromInfo, to)
}

func (f *Folder) IsIdentical(a, b os.FileInfo, aPath, bPath string) (bool, error) {
	if a.Name() != b.Name() {
		return false, nil
	}

	return compareAttributes(a, b), nil
}

type RegularFile struct{}

func (f *RegularFile) SupportsFile(info os.FileInfo) bool {
	return info.Mode().IsRegular()
}

func (f *RegularFile) Copy(fromInfo os.FileInfo, from, to string) error {
	fromFile, err := os.OpenFile(from, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("can't open file for reading: %w", err)
	}
	defer fromFile.Close()

	toFile, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fromInfo.Mode())
	if err != nil {
		return fmt.Errorf("can't create file: %w", err)
	}
	defer toFile.Close()

	_, err = io.Copy(toFile, fromFile)
	if err != nil {
		return fmt.Errorf("can't copy data: %w", err)
	}

	return nil
}

func (f *RegularFile) CopyAttributes(fromInfo os.FileInfo, to string) error {
	return copyAttributes(fromInfo, to)
}

func (f *RegularFile) IsIdentical(a, b os.FileInfo, aPath, bPath string) (bool, error) {
	if a.Name() != b.Name() {
		return false, nil
	}

	if !compareAttributes(a, b) {
		return false, nil
	}

	checkA, err := calculateChecksum(aPath)
	if err != nil {
		return false, err
	}
	checkB, err := calculateChecksum(bPath)
	if err != nil {
		return false, err
	}

	return checkA == checkB, nil
}

type CharDeviceFile struct{}

func (f *CharDeviceFile) SupportsFile(info os.FileInfo) bool {
	return info.Mode()&os.ModeCharDevice != 0
}

func (f *CharDeviceFile) Copy(fromInfo os.FileInfo, from, to string) error {
	fromInfoUnix := fromInfo.Sys().(*syscall.Stat_t)

	err := syscall.Mknod(to, fromInfoUnix.Mode, int(fromInfoUnix.Rdev))
	if err != nil {
		return fmt.Errorf("can't create character special file: %w", err)
	}

	return nil
}

func (f *CharDeviceFile) CopyAttributes(fromInfo os.FileInfo, to string) error {
	return copyAttributes(fromInfo, to)
}

func (f *CharDeviceFile) IsIdentical(a, b os.FileInfo, aPath, bPath string) (bool, error) {
	if a.Name() != b.Name() {
		return false, nil
	}

	if !compareAttributes(a, b) {
		return false, nil
	}

	aInfoUnix := a.Sys().(*syscall.Stat_t)
	bInfoUnix := b.Sys().(*syscall.Stat_t)

	return aInfoUnix.Rdev == bInfoUnix.Rdev, nil
}

var allATime = time.Now()

func copyAttributes(fromInfo os.FileInfo, to string) error {
	fromInfoUnix := fromInfo.Sys().(*syscall.Stat_t)

	err := syscall.Chown(to, int(fromInfoUnix.Uid), int(fromInfoUnix.Gid))
	if err != nil {
		return fmt.Errorf("can't change owner: %w", err)
	}

	err = syscall.Chmod(to, fromInfoUnix.Mode)
	if err != nil {
		return fmt.Errorf("can't change permissions: %w", err)
	}

	err = os.Chtimes(to, allATime, fromInfo.ModTime())
	if err != nil {
		return fmt.Errorf("can't change mod time: %w", err)
	}

	return nil
}

func compareAttributes(a, b os.FileInfo) bool {
	aPerm := a.Mode().Perm()
	bPerm := b.Mode().Perm()
	aSysMode := a.Sys().(*syscall.Stat_t)
	bSysMode := b.Sys().(*syscall.Stat_t)
	aUid := aSysMode.Uid
	bUid := bSysMode.Uid
	aGid := aSysMode.Gid
	bGid := bSysMode.Gid

	return aPerm == bPerm && aUid == bUid && aGid == bGid
}

func calculateChecksum(file string) (uint32, error) {
	calculator := crc32.NewIEEE()
	osFile, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(calculator, osFile); err != nil {
		return 0, err
	}
	return calculator.Sum32(), nil
}
