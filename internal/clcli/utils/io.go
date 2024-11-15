package clcli

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
	"os"
)

func CalculateChecksums(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha1.New()

	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil)), nil
}
