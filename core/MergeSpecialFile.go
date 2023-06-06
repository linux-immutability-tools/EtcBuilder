package core

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"os"
)

func MergeSpecialFile(user os.File, old os.File, new os.File) error {
	text1 := "_sndio:*:702:702:sndio privsep:/var/empty:/usr/sbin/nologin\ncyrus:*:60:60:the cyrus mail server:/nonexistent:/usr/sbin/nologin\nwebcamd:*:145:145:Webcamd user:/var/empty:/usr/sbin/nologin"
	text2 := "pulse:*:563:563:PulseAudio System User:/nonexistent:/usr/sbin/nologin\n_sndio:*:702:702:sndio privsep:/var/empty:/usr/sbin/nologin\ncyrus:*:60:60:the cyrus mail server:/nonexistent:/usr/sbin/nologin\nwebcamd:*:145:145:Webcamd user:/var/empty:/usr/sbin/nologin"
	text3 := "git_daemon:*:964:964:git daemon:/nonexistent:/usr/sbin/nologin\n_sndio:*:702:702:sndio privsep:/var/empty:/usr/sbin/nologin\ncyrus:*:60:60:the cyrus mail server:/nonexistent:/usr/sbin/nologin\nwebcamd:*:145:145:Webcamd user:/var/empty:/usr/sbin/nologin"

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, false)
	patches := dmp.PatchMake(text1, diffs)

	patchString := dmp.PatchToText(patches)

	fmt.Println("Old: \n" + text1)
	fmt.Println()
	fmt.Println("User: \n" + text2)
	fmt.Println()
	fmt.Println("New: \n" + text3)
	fmt.Println()
	fmt.Println("Patch: \n" + patchString)

	fmt.Println()
	result, _ := dmp.PatchApply(patches, text3)
	fmt.Println("Built: \n" + result)

	return nil
}
