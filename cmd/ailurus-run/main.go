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

	codeContents, err := io.ReadFile(cfg.configFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	machineParams, journalParams, err := parseMachineConfig(configContents)

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	machine := vm.New()

	fmt.Println(machineParams)
	fmt.Println(journalParams)

}
