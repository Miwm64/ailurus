package io

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadFile(path string) (string, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}

	return string(dat), nil
}

func ReadBytes(path string) ([]byte, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	return dat, nil
}

func WriteFile(path string, data string) error {
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}

	data = append(data, '\n')
	err = os.WriteFile(path, data, 0o644)

	return fmt.Errorf("write: %w", err)
}
