package cmd_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/linux-immutability-tools/EtcBuilder/cmd"
	"github.com/linux-immutability-tools/EtcBuilder/settings"
)

func prepareTestEnv(t *testing.T) (oldSys, newSys, oldUser, newUser string) {
	tmpDir := t.TempDir()
	oldSys = filepath.Join(tmpDir, "old Sys")
	newSys = filepath.Join(tmpDir, "new Sys")
	oldUser = filepath.Join(tmpDir, "old User")
	newUser = filepath.Join(tmpDir, "new User")
	err := os.Mkdir(oldSys, 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	err = os.Mkdir(newSys, 0o765)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	err = os.Mkdir(oldUser, 0o701)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	err = os.Mkdir(newUser, 0o700)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	return
}

func TestEmpty(t *testing.T) {
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	err := cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Error(err)
	}
}

func TestCleanDir(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	fileRel := "some/path to/file/my file.abc"
	myFile := filepath.Join(newUser, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.WriteFile(myFile, []byte("Some data"), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	_, err = os.Stat(filepath.Dir(myFile))
	if !os.IsNotExist(err) {
		t.Error("file was not removed successfully")
		t.Fail()
	}
}

func TestRegularFile(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	fileRel := "some/path to/file/my file.abc"
	myFile := filepath.Join(oldUser, fileRel)
	dirPerms := 0o751
	err = os.MkdirAll(filepath.Dir(myFile), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	filePerm := 0o715
	err = os.WriteFile(myFile, []byte("Some data"), fs.FileMode(filePerm))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	secondParentDir := filepath.Dir(filepath.Dir(filepath.Join(newUser, fileRel)))
	secondParentDirInfo, err := os.Stat(secondParentDir)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if int(secondParentDirInfo.Mode().Perm()) != dirPerms {
		t.Logf("Permissions %o instead of %o", secondParentDirInfo.Mode().Perm(), dirPerms)
		t.Log("Parent of parent dir doesn't have the right permissions")
		t.Fail()
	}

	newUserFile := filepath.Join(newUser, fileRel)
	contents, err := os.ReadFile(newUserFile)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if info, err := os.Lstat(newUserFile); err != nil || int(info.Mode().Perm()) != filePerm {
		t.Logf("Permissions %o instead of %o", info.Mode().Perm(), filePerm)
		t.Log("Permissions don't match")
		t.Fail()
	}
	if string(contents) != "Some data" {
		t.Log("Files did not match")
		t.FailNow()
	}
}

func TestSymlink(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	fileRel := "some/path to/link/my link"
	sym := "../some/ra ndom/path"
	myFile := filepath.Join(oldUser, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile), 0o753)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.Symlink(sym, myFile)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	err = cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	contents, err := os.Readlink(filepath.Join(newUser, fileRel))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if string(contents) != sym {
		t.Log("Files did not match")
		t.FailNow()
	}
}

func TestSpecialFile(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	fileRel := "test/äu/path/special file"

	oldUserFile := filepath.Join(oldUser, fileRel)
	err = os.MkdirAll(filepath.Dir(oldUserFile), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	oldSysFile := filepath.Join(oldSys, fileRel)
	err = os.MkdirAll(filepath.Dir(oldSysFile), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	newSysFile := filepath.Join(newSys, fileRel)
	err = os.MkdirAll(filepath.Dir(newSysFile), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	settings.SpecialFiles = append(settings.SpecialFiles, fileRel)

	part1 := `dnsmasq:x:993:991:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/usr/sbin/nologin`
	part2 := `colord:x:992:990:User for colord:/var/lib/colord:/sbin/nologin`
	part3 := `abrt:x:173:173::/etc/abrt:/sbin/nologin`
	part4 := `bin:x:1:1:bin:/bin:/sbin/nologin`
	part5 := `flatpak:x:986:984:Flatpak system helper:/:/usr/sbin/nologin`
	part6 := `rpc:x:32:32:Rpcbind Daemon:/var/lib/rpcbind:/sbin/nologin`
	part7 := `mail:x:8:12:mail:/var/spool/mail:/sbin/nologin`

	oldUserContents := part1 + "\n" + part2 + "\n" + part3 + "\n" + part4 + "\n" + part5 + "\n"

	oldSysContents := part1 + "\n" + part6 + "\n" + part2 + "\n" + part3 + "\n" + part5

	newSysContents := part1 + "\n" + part7 + "\n" + part6 + "\n" + part3 + "\n" + part5 + "\n"

	mergedContent := part1 + "\n" + part7 + "\n" + part3 + "\n" + part5 + "\n" + part4

	err = os.WriteFile(oldUserFile, []byte(oldUserContents), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.WriteFile(oldSysFile, []byte(oldSysContents), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.WriteFile(newSysFile, []byte(newSysContents), 0o777)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	contents, err := os.ReadFile(filepath.Join(newUser, fileRel))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if strings.TrimSpace(string(contents)) != strings.TrimSpace(mergedContent) {
		t.Log("Files did not get merged correctly")
		t.FailNow()
	}
}

func TestCharSpecial(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := prepareTestEnv(t)

	fileRel := "some/path to/link/my char special"
	myFile := filepath.Join(oldUser, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile), 0o753)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = syscall.Mknod(myFile, 0x2000, 0)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	err = cmd.ExtBuildCommand(oldSys, newSys, oldUser, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	var info syscall.Stat_t
	err = syscall.Lstat(filepath.Join(newUser, fileRel), &info)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if info.Mode != 0x2000 {
		t.Log("Character special was not created correctly")
		t.Fail()
	}
	if info.Rdev != 0 {
		t.Log("Device was not set correctly")
		t.Fail()
	}
}
