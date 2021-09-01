package util

import (
	"os"
	"strings"
)

func IsDirExist(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		if isNotExistErr(err) {
			return false, nil
		}

		return false, err
	}

	return fileInfo.IsDir(), nil
}

func IsRegularFileExist(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		if isNotExistErr(err) {
			return false, nil
		}

		return false, err
	}

	return fileInfo.Mode().IsRegular(), nil
}

func isNotExistErr(err error) bool {
	return os.IsNotExist(err) || IsNotDirectoryErr(err)
}

func IsNotDirectoryErr(err error) bool {
	return strings.HasSuffix(err.Error(), "not a directory")
}
