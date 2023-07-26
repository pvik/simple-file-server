package main

import (
	"os"
	"sort"
	"strings"
)

func getDirContent(path string) ([]os.FileInfo, error) {
	var files []os.FileInfo

	dir, err := os.Open(path)
	if err != nil {
		return files, err
	}

	defer dir.Close()
	files, err = dir.Readdir(-1)
	if err != nil {
		return files, err
	}

	// sort the files
	sort.SliceStable(files, func(i, j int) bool {
		if strings.Compare(files[i].Name(), files[j].Name()) < 0 {
			return true
		}
		return false

	})

	return files, nil

}
