package core

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"os"
)

func MergeSpecialFile(user string, old string, new string) error {

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

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(oldData), string(userData), false)
	patches := dmp.PatchMake(string(oldData), diffs)

	patchString := dmp.PatchToText(patches)

	fmt.Println("Old: \n" + string(oldData))
	fmt.Println()
	fmt.Println("User: \n" + string(userData))
	fmt.Println()
	fmt.Println("New: \n" + string(newData))
	fmt.Println()
	fmt.Println("Patch: \n" + patchString)

	fmt.Println()
	result, _ := dmp.PatchApply(patches, string(newData))
	fmt.Println("Built: \n" + result)

	return nil
}
