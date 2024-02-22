package util

import (
	"crypto/md5"
	"io"
	"os"
)

func Md5HashOfFile(filePath string) ([]byte, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new hasher
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	// hasher.Sum(nil) computes the hash of the file and returns it as a byte slice
	return hasher.Sum(nil), nil
}
