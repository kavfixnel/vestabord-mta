package main

import (
	"fmt"
	"os"
	"strings"
)

func loadEnvVar(path, name string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		if strings.TrimSpace(key) == name {
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			if value == "" {
				return "", fmt.Errorf("%s is empty in %s", name, path)
			}
			return value, nil
		}
	}

	return "", fmt.Errorf("%s not found in %s", name, path)
}
