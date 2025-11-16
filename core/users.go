package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type PasswdEntry struct {
	Name      string
	Password  string
	Uid       int
	Gid       int
	Gecos     string
	Directory string
	Shell     string
}

func NewPasswdFile(path string) (*PasswdFile, error) {
	passFile := PasswdFile{Filepath: path}
	passFile.Contents = make(map[string]PasswdEntry)
	err := passFile.parse()
	if err != nil {
		return nil, fmt.Errorf("can't parse file: %w", err)
	}
	return &passFile, nil
}

type PasswdFile struct {
	Filepath string
	Contents map[string]PasswdEntry
}

func (e *PasswdFile) WriteToFile(path string) error {
	lines := make(map[int]string)
	uids := []int{}

	for _, entry := range e.Contents {
		line := entry.Name + ":" +
			entry.Password + ":" +
			strconv.Itoa(entry.Uid) + ":" +
			strconv.Itoa(entry.Gid) + ":" +
			entry.Gecos + ":" +
			entry.Directory + ":" +
			entry.Shell + "\n"

		lines[entry.Uid] = line
		uids = append(uids, entry.Uid)
	}

	slices.Sort(uids)

	fileContent := ""

	for _, uid := range uids {
		fileContent += lines[uid]
	}

	err := os.WriteFile(path, []byte(fileContent), 0o644)
	if err != nil {
		return fmt.Errorf("can't write file: %w", err)
	}

	return nil
}

func (e *PasswdFile) parse() error {
	passwdContents, err := os.ReadFile(e.Filepath)
	if err != nil {
		return fmt.Errorf("can't read file: %w", err)
	}

	for line := range strings.SplitAfterSeq(string(passwdContents), "\n") {
		line := strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			continue
		}

		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}

		entry := PasswdEntry{Name: fields[0], Password: fields[1], Uid: uid, Gid: gid, Gecos: fields[4], Directory: fields[5], Shell: fields[6]}

		e.Contents[entry.Name] = entry
	}

	return nil
}

const LowestSystemUid = 101
const HighestSystemUid = 999

var ErrNoUidsLeft = errors.New("All available UIDs are taken")

func (e *PasswdFile) AddSystemUser(name string, gid int, requestUid int, password string, gecos string, directory string, shell string) (int, error) {
	if existing, alreadyExists := e.Contents[name]; alreadyExists {
		return existing.Uid, nil
	}

	uidExists := make(map[int]bool)

	for _, value := range e.Contents {
		uidExists[value.Uid] = true
	}

	entry := PasswdEntry{Name: name, Uid: requestUid, Gid: gid, Password: password, Gecos: gecos, Directory: directory, Shell: shell}

	if !uidExists[requestUid] {
		e.Contents[name] = entry

		return requestUid, nil
	}

	for uid := HighestSystemUid; uid >= LowestSystemUid; uid-- {
		if uidExists[uid] {
			continue
		}

		entry.Uid = uid

		e.Contents[name] = entry

		return uid, nil
	}

	return -1, ErrNoUidsLeft
}

func (e *PasswdFile) MergeWithOther(other PasswdFile, groupMapping map[int]int, nogroupID int) []error {
	errList := []error{}

	for name, entry := range other.Contents {
		if _, exists := e.Contents[name]; exists {
			continue
		}

		gid, ok := groupMapping[entry.Gid]
		if !ok {
			gid = nogroupID
		}

		_, err := e.AddSystemUser(name, gid, entry.Uid, entry.Password, entry.Gecos, entry.Directory, entry.Shell)
		if err != nil {
			errList = append(errList, err)
			continue
		}
	}

	return errList

}

func CreateUserMapping(from, to PasswdFile) (map[int]int, error) {
	mapping := make(map[int]int)

	for key, val := range from.Contents {
		to, ok := to.Contents[key]
		if !ok {
			return nil, errors.New("can't find " + key + " in other map")
		}
		mapping[val.Uid] = to.Uid
	}

	return mapping, nil
}

// MergeInShadow merges extra entries from the shadow file in extraShadowDir into the shadow file in shadowDir
//
// returns the number of added lines and and errors for reading or writing files
func MergeInShadow(shadowDir string, extraShadowDir string) (int, error) {
	shadowFilePath := filepath.Join(shadowDir, "shadow")
	extraShadowFilePath := filepath.Join(extraShadowDir, "shadow")

	shadowFileContents, err := os.ReadFile(shadowFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open shadow file: %w", err)
	}
	extraShadowFileContents, err := os.ReadFile(extraShadowFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open extra shadow file: %w", err)
	}

	shadowEntries := make(map[string]string)
	for line := range strings.SplitAfterSeq(string(shadowFileContents), "\n") {
		line = strings.TrimSpace(line)
		name, info, _ := strings.Cut(line, ":")
		if name == "" {
			continue
		}

		shadowEntries[name] = info
	}

	newLines := 0

	for line := range strings.SplitAfterSeq(string(extraShadowFileContents), "\n") {
		line = strings.TrimSpace(line)
		name, info, _ := strings.Cut(line, ":")
		if name == "" {
			continue
		}

		_, ok := shadowEntries[name]
		if !ok {
			shadowEntries[name] = info
			newLines += 1
		}
	}

	if newLines == 0 {
		return 0, nil
	}

	newFileContents := ""

	for name, info := range shadowEntries {
		newFileContents += name + ":" + info + "\n"
	}

	err = os.WriteFile(shadowFilePath, []byte(newFileContents), 0o640)
	if err != nil {
		return 0, fmt.Errorf("can't write shadow file: %w", err)
	}

	return newLines, nil
}
