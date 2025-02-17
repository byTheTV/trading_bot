package utils

import (
	"fmt"
	"os"
	"strings"
)

func LoadPairs(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var pairs []string
	for _, line := range lines {
		pair := strings.TrimSpace(line)
		if pair != "" {
			pairs = append(pairs, pair)
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("file %s is empty or contains no pairs", filename)
	}

	return pairs, nil
}
