package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func ApplyOwnerMappingRecursive(dir string, uidMapping map[int]int, gidMapping map[int]int) error {
	return applyOwnerMappingRecursive(dir, uidMapping, gidMapping, syscall.Chown)
}

func applyOwnerMappingRecursive(dir string, uidMapping map[int]int, gidMapping map[int]int, chownFn func(string, int, int) error) error {
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("can't search path \"%s\": %w", path, err)
		}

		err = applyOwnerMapping(filepath.Join(dir, path), uidMapping, gidMapping, chownFn)
		if err != nil {
			return fmt.Errorf("can't apply ownership of %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("can't copy all files: %w", err)
	}

	return nil
}

func ApplyOwnerMapping(path string, uidMapping map[int]int, gidMapping map[int]int) error {
	return applyOwnerMapping(path, uidMapping, gidMapping, syscall.Chown)
}

func applyOwnerMapping(path string, uidMapping map[int]int, gidMapping map[int]int, chownFn func(string, int, int) error) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("can't get info about file: %w", err)
	}

	if isSymlink(info) {
		return nil
	}

	infoUnix := info.Sys().(*syscall.Stat_t)

	oldUid := int(infoUnix.Uid)
	newUid, ok := uidMapping[oldUid]
	if !ok {
		oldUid = newUid
	}

	oldGid := int(infoUnix.Gid)
	newGid, ok := gidMapping[oldGid]
	if !ok {
		newGid = oldGid
	}

	if newUid == oldUid && newGid == oldGid {
		return nil
	}

	fmt.Println("changing ownership of:", path)

	err = chownFn(path, newUid, newGid)
	if err != nil {
		return fmt.Errorf("can't change owner: %w", err)
	}

	return nil
}

func isSymlink(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
