package main

import (
	"errors"
	"fmt"
	"os"
)

// ensurePath checks if a path exists and is a directory, if not it creates the directory. Returns true if the directories were created.
func ensurePath(path string) (bool, error) {
	// Check if output directory exists, if not create and perform extraction
	targetInfo, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if err = os.MkdirAll(path, 0o755); err != nil {
			return false, err
		}
		return true, nil
	}
	if !targetInfo.IsDir() {
		return false, fmt.Errorf("%s isn't a directory", path)
	}
	return false, nil
}
