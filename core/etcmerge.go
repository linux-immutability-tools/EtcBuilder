package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type ErrMergeFiles struct {
	msg  string
	errs []error
}

func (e *ErrMergeFiles) Error() string {
	return e.msg + ": " + fmt.Sprint(e.errs)
}

func (e *ErrMergeFiles) Unwrap() []error {
	return e.errs
}

// BuildNewEtc fixes the owner of the new lower etc folder and create the new upper etc folder
func BuildNewEtc(lowerOld, upperOld, lowerNew, upperNew string) error {

	os.RemoveAll(upperNew)
	os.MkdirAll(lowerOld, 0x755)
	os.MkdirAll(upperOld, 0x755)
	os.MkdirAll(lowerNew, 0x755)

	err := CarbonCopyRecursive(upperOld, upperNew)
	if err != nil {
		return fmt.Errorf("can't create new upper etc: %w", err)
	}

	groupFile, groupMapping, err := handleGroupFiles(upperOld, lowerNew, upperNew)
	if err != nil {
		return err
	}

	_, err = MergeInGshadow(upperNew, lowerNew)
	if err != nil {
		return fmt.Errorf("can't merge lower gshadow file into upper: %w", err)
	}

	_, userMapping, err := handlePasswdFiles(upperOld, lowerNew, upperNew, groupFile, groupMapping)
	if err != nil {
		return err
	}

	_, err = MergeInShadow(upperNew, lowerNew)
	if err != nil {
		return fmt.Errorf("can't merge lower shadow file into upper: %w", err)
	}

	_, err = MergeInShells(upperNew, lowerNew)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("can't merge lower shells file into upper: %w", err)
	}

	err = ApplyOwnerMappingRecursive(lowerNew, userMapping, groupMapping)
	if err != nil {
		return fmt.Errorf("can't apply owner mapping: %w", err)
	}

	RemoveIdenticalFiles(upperNew, lowerNew)

	return nil
}

func handleGroupFiles(upperOld, lowerNew, upperNew string) (*GroupFile, map[int]int, error) {
	groupFile, err := NewGroupFile(filepath.Join(upperOld, "group"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't open current group file: %w", err)
	}

	newLowerGroupFile, err := NewGroupFile(filepath.Join(lowerNew, "group"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't open new lower group file: %w", err)
	}

	errs := groupFile.MergeWithOther(*newLowerGroupFile)
	if len(errs) != 0 {
		return nil, nil, &ErrMergeFiles{msg: "can't merge groups", errs: errs}
	}

	err = groupFile.WriteToFile(filepath.Join(upperNew, "group"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't write merged group file: %w", err)
	}

	groupMapping, err := CreateGroupMapping(*newLowerGroupFile, *groupFile)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create group mapping: %w", err)
	}

	return groupFile, groupMapping, nil
}

func handlePasswdFiles(upperOld, lowerNew, upperNew string, groupFile *GroupFile, groupMapping map[int]int) (*PasswdFile, map[int]int, error) {
	passwdFile, err := NewPasswdFile(filepath.Join(upperOld, "passwd"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't open current passwd file: %w", err)
	}

	newLowerPasswdFile, err := NewPasswdFile(filepath.Join(lowerNew, "passwd"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't open new lower passwd file: %w", err)
	}

	var nogroupGid int
	if nogroup, ok := groupFile.Contents["nogroup"]; ok {
		nogroupGid = nogroup.Gid
	} else {
		nogroupGid = 65534
	}

	errs := passwdFile.MergeWithOther(*newLowerPasswdFile, groupMapping, nogroupGid)
	if len(errs) != 0 {
		return nil, nil, &ErrMergeFiles{msg: "can't merge users", errs: errs}
	}

	err = passwdFile.WriteToFile(filepath.Join(upperNew, "passwd"))
	if err != nil {
		return nil, nil, fmt.Errorf("can't write merged passwd file: %w", err)
	}

	userMapping, err := CreateUserMapping(*newLowerPasswdFile, *passwdFile)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create user mapping: %w", err)
	}

	return passwdFile, userMapping, nil
}

// MergeInShells merges extra entries from the shells file in extraShellsDir into the shells file in shellsDir
func MergeInShells(shellsDir, extraShellsDir string) (int, error) {
	shellsFilePath := filepath.Join(shellsDir, "shells")
	extraShellsFilePath := filepath.Join(extraShellsDir, "shells")

	shellsFileContents, err := os.ReadFile(shellsFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open shells file: %w", err)
	}
	extraShellsFileContents, err := os.ReadFile(extraShellsFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open extra shells file: %w", err)
	}

	shellsList := []string{}

	for line := range strings.SplitSeq(string(shellsFileContents), "\n") {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		shellsList = append(shellsList, line)
	}

	for index, line := range shellsList {
		line = strings.TrimSpace(line)
		shellsList[index] = line
	}

	addedCount := 0

	for extraLine := range strings.SplitSeq(string(extraShellsFileContents), "\n") {
		extraLine := strings.TrimSpace(extraLine)
		if extraLine == "" {
			continue
		}
		if strings.HasPrefix(extraLine, "#") {
			continue
		}
		if slices.Contains(shellsList, extraLine) {
			continue
		}

		shellsList = append(shellsList, extraLine)
		addedCount++
	}

	mergedFileContents := strings.Join(shellsList, "\n") + "\n"

	os.WriteFile(shellsFilePath, []byte(mergedFileContents), 0o644)

	return addedCount, nil
}
