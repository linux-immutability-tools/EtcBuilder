package core

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"
)

func KeepUserFile(user os.File, new os.File) (bool, error) {

	userFileHash, err := calculateHash(&user)
	if err != nil {
		return true, fmt.Errorf("failed to calculate hash of user file")
	}

	newFilehash, err := calculateHash(&new)
	if err != nil {
		return true, fmt.Errorf("failed to calculate hash of new file")
	}

	if strings.Compare(strings.TrimSpace(userFileHash), strings.TrimSpace(newFilehash)) != 0 {
		return true, nil
	}

	return false, nil
}

func calculateHash(file *os.File) (string, error) {
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:20]
	return strings.TrimSpace(fmt.Sprintf("%x", hashInBytes)), nil
}
