package main

import (
	"fmt"
	"os"
)

func keysFromFiles(paths ...string) ([]string, error) {
	keys := make([]string, len(paths))
	for i, path := range paths {
		keyBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		keys[i] = fmt.Sprintf("user%d:%s", i, string(keyBytes))
	}
	return keys, nil
}
