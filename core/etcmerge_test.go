package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

const passwdLowerOld = `
root:x:0:0:root:/root:/bin/bash
irc:x:39:39:ircd:/run/ircd:/usr/sbin/nologin
_apt:x:42:65534::/nonexistent:/usr/sbin/nologin
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
`

const passwdUpperOld = `
root:x:0:0:root:/root:/bin/bash
irc:x:39:39:ircd:/run/ircd:/usr/sbin/nologin
_apt:x:42:65534::/nonexistent:/usr/sbin/nologin
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
test::1000:1000:Tau:/home/test:/usr/bin/bash
`

const passwdLowerNew = `
root:x:0:0:root:/root:/bin/bash
irc:x:39:39:ircd:/run/ircd:/usr/sbin/nologin
uucp:x:10:10:uucp:/var/spool/uucp:/usr/sbin/nologin
_apt:x:42:65534::/nonexistent:/usr/sbin/nologin
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
`

const passwdExpect = `
root:x:0:0:root:/root:/bin/bash
uucp:x:10:10:uucp:/var/spool/uucp:/usr/sbin/nologin
irc:x:39:39:ircd:/run/ircd:/usr/sbin/nologin
_apt:x:42:65534::/nonexistent:/usr/sbin/nologin
test::1000:1000:Tau:/home/test:/usr/bin/bash
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
`

const groupLowerOld = `
root:x:0:
irc:x:39:
nogroup:x:65534:
`

const groupUpperOld = `
root:x:0:
irc:x:39:
nogroup:x:65534:
test:x:1000:
`

const groupLowerNew = `
root:x:0:
irc:x:39:
uucp:x:10:
nogroup:x:65534:
`

const groupExpect = `
root:x:0:
uucp:x:10:
irc:x:39:
test:x:1000:
nogroup:x:65534:
`

const gshadowLowerOld = `
root:*::
irc:*::
nogroup:*::
`

const gshadowUpperOld = `
root:*::
irc:*::
nogroup:*::
test:!::
`

const gshadowLowerNew = `
root:*::
uucp:*::
irc:*::
nogroup:*::
`

const gshadowExpect = `
uucp:*::
root:*::
irc:*::
nogroup:*::
test:!::
`

const shadowLowerOld = `
root::20248:0:99999:7:::
irc:*:20228:0:99999:7:::
nobody:*:20228:0:99999:7:::
`
const shadowUpperOld = `
root::20248:0:99999:7:::
irc:*:20228:0:99999:7:::
nobody:*:20228:0:99999:7:::
test:$j$jjT$huf789w.$iojfw3897:20191:0:99999:7:::
`

const shadowLowerNew = `
root::20248:0:99999:7:::
uucp:*:20228:0:99999:7:::
irc:*:20228:0:99999:7:::
nobody:*:20228:0:99999:7:::
`

const shadowExpect = `
nobody:*:20228:0:99999:7:::
test:$j$jjT$huf789w.$iojfw3897:20191:0:99999:7:::
uucp:*:20228:0:99999:7:::
root::20248:0:99999:7:::
irc:*:20228:0:99999:7:::
`

const shellsLowerOld = `
# /etc/shells: valid login shells
/bin/sh
/usr/bin/sh
/bin/bash
`
const shellsUpperOld = `
# /etc/shells: valid login shells
/bin/sh
/usr/bin/sh
/bin/bash
/usr/bin/fish
`

const shellsLowerNew = `
# /etc/shells: valid login shells
/bin/sh
/usr/bin/sh
/bin/bash
/usr/bin/vso-os-shell
`

const shellsExpect = `
# /etc/shells: valid login shells
/bin/sh
/usr/bin/sh
/bin/bash
/usr/bin/fish
/usr/bin/vso-os-shell
`

func setupEnvironment(t *testing.T) (string, string, string, string) {
	testEtcPath := t.TempDir()

	lowerOld := filepath.Join(testEtcPath, "lowerOld")
	upperOld := filepath.Join(testEtcPath, "upperOld")
	lowerNew := filepath.Join(testEtcPath, "lowerNew")
	upperNew := filepath.Join(testEtcPath, "upperNew")

	os.RemoveAll(testEtcPath)

	err := os.MkdirAll(lowerOld, 0o755)
	if err != nil {
		t.Error(err)
	}
	err = os.MkdirAll(upperOld, 0o755)
	if err != nil {
		t.Error(err)
	}
	err = os.MkdirAll(lowerNew, 0o755)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerOld, "passwd"), []byte(passwdLowerOld), 0o644)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(upperOld, "passwd"), []byte(passwdUpperOld), 0o644)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerNew, "passwd"), []byte(passwdLowerNew), 0o644)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(filepath.Join(lowerOld, "group"), []byte(groupLowerOld), 0o644)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(upperOld, "group"), []byte(groupUpperOld), 0o644)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerNew, "group"), []byte(groupLowerNew), 0o644)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(filepath.Join(lowerOld, "gshadow"), []byte(gshadowLowerOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(upperOld, "gshadow"), []byte(gshadowUpperOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerNew, "gshadow"), []byte(gshadowLowerNew), 0o640)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(filepath.Join(lowerOld, "shadow"), []byte(shadowLowerOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(upperOld, "shadow"), []byte(shadowUpperOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerNew, "shadow"), []byte(shadowLowerNew), 0o640)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(filepath.Join(lowerOld, "shells"), []byte(shellsLowerOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(upperOld, "shells"), []byte(shellsUpperOld), 0o640)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(lowerNew, "shells"), []byte(shellsLowerNew), 0o640)
	if err != nil {
		t.Error(err)
	}

	return lowerOld, lowerNew, upperOld, upperNew
}

func TestEmpty(t *testing.T) {
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

	err := BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Error(err)
	}
}

func TestCleanDir(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
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
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
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
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
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

func TestSpecialFiles(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	allSpecials := map[string]string{"passwd": passwdExpect, "shadow": shadowExpect, "group": groupExpect, "gshadow": gshadowExpect, "shells": shellsExpect}

	for special, expect := range allSpecials {
		contents, err := os.ReadFile(filepath.Join(newUser, special))
		if err != nil {
			t.Log(err)
			t.FailNow()
		}

		err = compareSpecialContents(string(contents), expect)
		if err != nil {
			t.Fatal(special, "did not get merged correctly:", err)
		}
	}

}

func compareSpecialContents(a, b string) error {

	aParts := strings.Split(strings.TrimSpace(a), "\n")
	bParts := strings.Split(strings.TrimSpace(b), "\n")

	if len(aParts) != len(bParts) {
		return errors.New("number of entries doesn't match")
	}

	for _, aPart := range aParts {
		if !slices.Contains(bParts, aPart) {
			return errors.New("b is missing line" + aPart)
		}
	}

	for _, bPart := range bParts {
		if !slices.Contains(aParts, bPart) {
			return errors.New("a is missing line" + bPart)
		}
	}

	return nil
}

func TestCharSpecial(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
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

func TestCleanup(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	myFile2 := filepath.Join(newSys, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile2), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.WriteFile(myFile2, []byte("Some data"), fs.FileMode(filePerm))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	newUserFile := filepath.Join(newUser, fileRel)

	_, err = os.Lstat(newUserFile)

	if err == nil {
		t.Fatal("identical file was not cleaned up")
	}
}

func TestCleanup2(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	myFile2 := filepath.Join(newSys, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile2), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.WriteFile(myFile2, []byte("Some other data"), fs.FileMode(filePerm))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	newUserFile := filepath.Join(newUser, fileRel)

	_, err = os.Lstat(newUserFile)

	if err != nil {
		t.Fatal("file was cleaned up even though it's not identical")
	}
}

func TestCleanup3(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

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

	myFile2 := filepath.Join(newSys, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile2), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	filePerm2 := 0o777
	err = os.WriteFile(myFile2, []byte("Some data"), fs.FileMode(filePerm2))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	newUserFile := filepath.Join(newUser, fileRel)

	_, err = os.Lstat(newUserFile)

	if err != nil {
		t.Fatal("file was cleaned up even though the attributes were not identical")
	}
}

func TestCleanupSymlink(t *testing.T) {
	var err error
	oldSys, newSys, oldUser, newUser := setupEnvironment(t)

	fileRel := "some/path to/file/my file.abc"
	myFile := filepath.Join(oldUser, fileRel)
	dirPerms := 0o751
	err = os.MkdirAll(filepath.Dir(myFile), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.Symlink("some/link", myFile)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	err = os.Symlink("some/link", myFile+"different")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	myFile2 := filepath.Join(newSys, fileRel)
	err = os.MkdirAll(filepath.Dir(myFile2), fs.FileMode(dirPerms))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = os.Symlink("some/link", myFile2)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	err = os.Symlink("some/other/link", myFile2+"different")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	err = BuildNewEtc(oldSys, oldUser, newSys, newUser)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	newUserFile := filepath.Join(newUser, fileRel)

	_, err = os.Lstat(newUserFile)

	if err == nil {
		t.Fatal("symlink was not cleaned up even though it was identical")
	}

	newUserFileDifferent := filepath.Join(newUser, fileRel+"different")

	_, err = os.Lstat(newUserFileDifferent)

	if err != nil {
		t.Fatal("symlink was cleaned up even though it was different")
	}
}
