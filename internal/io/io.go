package io

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadFile(path string) (string, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}

func WriteFile(path string, data string) error {
	err := os.WriteFile(path, []byte(data), 0644)
	return err
}

func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
