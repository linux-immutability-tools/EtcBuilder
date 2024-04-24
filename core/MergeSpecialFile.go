package core

import (
	"os"
	"slices"
	"strings"
)

func removeEmptyLines(lines []string) []string {
	newLines := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			newLines = append(newLines, line)
		}
	}
	return newLines
}

func createLineDiff(baseData, modifiedData string) (added, removed []string) {
	oldLines := strings.Split(baseData, "\n")
	newLines := strings.Split(modifiedData, "\n")
	oldLines = removeEmptyLines(oldLines)
	newLines = removeEmptyLines(newLines)

	for _, line := range oldLines {
		if !slices.Contains(newLines, line) {
			removed = append(removed, line)
		}
	}
	for _, line := range newLines {
		if !slices.Contains(oldLines, line) {
			added = append(added, line)
		}
	}

	return
}

func applyLineDiff(added, removed []string, data string) string {
	dataLines := strings.Split(data, "\n")
	dataLines = removeEmptyLines(dataLines)

	for _, line := range removed {
		pos := slices.Index(dataLines, line)
		if pos >= 0 {
			dataLines = slices.Delete(dataLines, pos, pos+1)
		}
	}
	for _, line := range added {
		if !slices.Contains(dataLines, line) {
			dataLines = append(dataLines, line)
		}
	}

	return strings.Join(dataLines, "\n") + "\n"
}

func MergeSpecialFile(user string, old string, new string, out string) error {
	// Merges special files
	// Files get merged by first forming a diff between the old file and the user file
	// Then applying the generated patch to the new file
	// The new file then gets written to the given destination
	userData, err := os.ReadFile(user)
	if err != nil {
		return err
	}
	oldData, err := os.ReadFile(old)
	if err != nil {
		return err
	}
	newData, err := os.ReadFile(new)
	if err != nil {
		return err
	}

	added, removed := createLineDiff(string(oldData), string(userData))

	result := applyLineDiff(added, removed, string(newData))
	filePerms, err := os.Stat(new)
	if err != nil {
		return err
	}
	err = os.WriteFile(out, []byte(result), filePerms.Mode())
	if err != nil {
		return err
	}
	return nil
}
