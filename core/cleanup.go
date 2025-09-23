package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Comparable interface {
	SupportsFile(info os.FileInfo) bool
	IsIdentical(a, b os.FileInfo, aPath, bPath string) (bool, error)
}

// no folders since checking if they are empty complicates things
var comparables = [...]Comparable{&RegularFile{}, &Symlink{}, &CharDeviceFile{}}

// RemoveIdenticalFiles removes files from target if an identical
// version exists in the same location in base.
func RemoveIdenticalFiles(target string, base string) {
	filesToRemove := []string{}

	err := fs.WalkDir(os.DirFS(target), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: can't search path \"%s\" for cleanup: %s", path, err)
		}

		baseFile := filepath.Join(base, path)
		targetFile := filepath.Join(target, path)

		targetInfo, err := os.Lstat(targetFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning:", err)
			return nil
		}

		baseInfo, err := os.Lstat(baseFile)
		if err != nil {
			// no base file, so keep target
			return nil
		}

		for _, comparable := range comparables {
			if comparable.SupportsFile(targetInfo) {
				isIdentical, err := comparable.IsIdentical(targetInfo, baseInfo, targetFile, baseFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Warning:", err)
					return nil
				}
				if isIdentical {
					filesToRemove = append(filesToRemove, targetFile)
				}
				return nil
			}
		}

		return nil
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning:", err)
		return
	}

	for _, toRemove := range filesToRemove {
		err := os.Remove(toRemove)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: can not remove unnecessary file"+toRemove+":", err)
		}
	}
}
