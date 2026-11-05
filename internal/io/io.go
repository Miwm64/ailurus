package io

import (
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
	t := []byte(data)
	fmt.Println(t)
	err := os.WriteFile(path, []byte(data), 0644)
	return err
}
