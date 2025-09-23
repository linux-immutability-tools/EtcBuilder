package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func CarbonCopyRecursive(from, to string) error {

	err := fs.WalkDir(os.DirFS(from), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("can't search path \"%s\": %w", path, err)
		}

		from := filepath.Join(from, path)
		to := filepath.Join(to, path)

		err = CarbonCopy(from, to)
		if err != nil {
			return fmt.Errorf("can't copy \"%s\": %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("can't copy all files: %w", err)
	}

	return nil
}

type Copyable interface {
	SupportsFile(info os.FileInfo) bool
	Copy(fromInfo os.FileInfo, from, to string) error
	CopyAttributes(fromInfo os.FileInfo, to string) error
}

var copyables = [...]Copyable{&Folder{}, &RegularFile{}, &Symlink{}, &CharDeviceFile{}}

var ErrUnsupportedFiletype = errors.New("unsupported file type")

func CarbonCopy(from, to string) error {
	fromInfo, err := os.Lstat(from)
	if err != nil {
		return fmt.Errorf("can't find information about file: %w", err)
	}

	for _, copyable := range copyables {
		if copyable.SupportsFile(fromInfo) {
			err = copyable.Copy(fromInfo, from, to)
			if err != nil {
				return fmt.Errorf("can't copy node: %w", err)
			}

			err = copyable.CopyAttributes(fromInfo, to)
			if err != nil {
				return fmt.Errorf("can't copy attributes: %w", err)
			}

			return nil
		}
	}

	return ErrUnsupportedFiletype
}
