package main

import (
	"fmt"
	"os"

	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/vm"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	configContents, err := io.ReadFile(cfg.configFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	codeContents, err := io.ReadBytes(cfg.configFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	machineParams, _, err := parseMachineConfig(configContents)

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	_, err = vm.New(codeContents, machineParams)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

}
