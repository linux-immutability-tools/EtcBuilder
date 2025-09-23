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

func CreateGroupMapping(from, to GroupFile) (map[int]int, error) {
	mapping := make(map[int]int)

	for key, val := range from.Contents {
		to, ok := to.Contents[key]
		if !ok {
			return nil, errors.New("can't find " + key + " in other map")
		}
		mapping[val.Gid] = to.Gid
	}

	return mapping, nil
}

type GroupEntry struct {
	Name     string
	Password string
	Gid      int
	Users    []string
}

func NewGroupFile(path string) (*GroupFile, error) {
	groupFile := GroupFile{Filepath: path}
	groupFile.Contents = make(map[string]GroupEntry)
	err := groupFile.parse()
	if err != nil {
		return nil, fmt.Errorf("can't parse group file: %w", err)
	}
	return &groupFile, nil
}

type GroupFile struct {
	Filepath string
	Contents map[string]GroupEntry
}

func (e *GroupFile) WriteToFile(path string) error {
	lines := make(map[int]string)
	gids := []int{}

	for _, entry := range e.Contents {
		line := entry.Name + ":" +
			entry.Password + ":" +
			strconv.Itoa(entry.Gid) + ":" +
			strings.Join(entry.Users, ",") + "\n"

		lines[entry.Gid] = line
		gids = append(gids, entry.Gid)
	}

	slices.Sort(gids)

	fileContent := ""

	for _, gid := range gids {
		fileContent += lines[gid]
	}

	err := os.WriteFile(path, []byte(fileContent), 0o644)
	if err != nil {
		return fmt.Errorf("can't write file: %w", err)
	}

	return nil
}

func (e *GroupFile) parse() error {
	groupContents, err := os.ReadFile(e.Filepath)
	if err != nil {
		return fmt.Errorf("can't read group file: %w", err)
	}

	for line := range strings.SplitAfterSeq(string(groupContents), "\n") {
		line := strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			continue
		}

		if fields[0] == "" {
			continue
		}

		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		users := strings.Split(fields[3], ",")

		if len(users) == 1 && users[0] == "" {
			users = []string{}
		}

		entry := GroupEntry{Name: fields[0], Password: fields[1], Gid: gid, Users: users}

		e.Contents[entry.Name] = entry
	}

	return nil
}

const LowestSystemGid = 101
const HighestSystemGid = 999

var ErrNoGidsLeft = errors.New("All available GIDs are taken")

func (e *GroupFile) AddSystemGroup(name string, requestGid int, password string, users []string) (int, error) {
	if existing, alreadyExists := e.Contents[name]; alreadyExists {
		return existing.Gid, nil
	}

	gidExists := make(map[int]bool)

	for _, value := range e.Contents {
		gidExists[value.Gid] = true
	}

	entry := GroupEntry{Name: name, Gid: requestGid, Password: password, Users: users}

	if !gidExists[requestGid] {
		e.Contents[name] = entry

		return requestGid, nil
	}

	for gid := HighestSystemGid; gid >= LowestSystemGid; gid-- {
		if gidExists[gid] {
			continue
		}

		entry.Gid = gid
		e.Contents[name] = entry

		return gid, nil
	}

	return -1, ErrNoGidsLeft
}

func (e *GroupFile) MergeWithOther(other GroupFile) []error {
	errList := []error{}

	for name, entry := range other.Contents {
		if _, exists := e.Contents[name]; exists {
			continue
		}

		_, err := e.AddSystemGroup(entry.Name, entry.Gid, entry.Password, entry.Users)
		if err != nil {
			errList = append(errList, err)
			continue
		}
	}

	return errList
}

// MergeInGshadow merges extra entries from the gshadow file in extraGshadowDir into the gshadow file in gshadowDir
//
// returns the number of added lines and and errors for reading or writing files
func MergeInGshadow(gshadowDir string, extraGshadowDir string) (int, error) {
	gshadowFilePath := filepath.Join(gshadowDir, "gshadow")
	extraGshadowFilePath := filepath.Join(extraGshadowDir, "gshadow")

	gshadowFileContents, err := os.ReadFile(gshadowFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open gshadow file: %w", err)
	}
	extraGshadowFileContents, err := os.ReadFile(extraGshadowFilePath)
	if err != nil {
		return 0, fmt.Errorf("can't open extra gshadow file: %w", err)
	}

	gshadowEntries := make(map[string]string)
	for line := range strings.SplitAfterSeq(string(gshadowFileContents), "\n") {
		line = strings.TrimSpace(line)
		name, info, _ := strings.Cut(line, ":")
		if name == "" {
			continue
		}

		gshadowEntries[name] = info
	}

	newLines := 0

	for line := range strings.SplitAfterSeq(string(extraGshadowFileContents), "\n") {
		line = strings.TrimSpace(line)
		name, info, _ := strings.Cut(line, ":")
		if name == "" {
			continue
		}

		_, ok := gshadowEntries[name]
		if !ok {
			gshadowEntries[name] = info
			newLines += 1
		}
	}

	if newLines == 0 {
		return 0, nil
	}

	newFileContents := ""

	for name, info := range gshadowEntries {
		newFileContents += name + ":" + info + "\n"
	}

	err = os.WriteFile(gshadowFilePath, []byte(newFileContents), 0o640)
	if err != nil {
		return 0, fmt.Errorf("can't write gshadow file: %w", err)
	}

	return newLines, nil
}
