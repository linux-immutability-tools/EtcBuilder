package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNoop(t *testing.T) {
	dir := t.TempDir()

	testfile := filepath.Join(dir, "some weird testfile.x")

	err := os.WriteFile(testfile, []byte("test content"), 0o743)
	if err != nil {
		t.Fatal(err)
	}

	err = applyOwnerMappingRecursive(dir, map[int]int{}, map[int]int{}, func(path string, uid, gid int) error {
		t.Fatal("changed file", path, "even though no mapping was set")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
