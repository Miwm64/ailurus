package main

import (
	"errors"
	"flag"
	"strings"
)

var (
	errNoInput     = errors.New("no input files")
	errBothModes   = errors.New("-c and -E are mutually exclusive")
	errEmptyDefine = errors.New("-D: empty name")
)

type config struct {
	inputFiles  []string
	outputFile  string
	listingFile string
	defines     map[string]string

	assembleOnly   bool
	preprocessOnly bool
	stdlib         bool
}

func parseConfig() (config, error) {
	cfg := config{defines: map[string]string{}}

	var noStdlib bool

	flag.StringVar(&cfg.outputFile, "o", "a.bin", "output `file`")
	flag.StringVar(&cfg.listingFile, "l", "", "debug listing `file`")
	flag.BoolVar(&cfg.assembleOnly, "c", false, "assemble only: one .o per input, no linking")
	flag.BoolVar(&cfg.preprocessOnly, "E", false, "preprocess only: print to stdout")
	flag.BoolVar(&noStdlib, "no-stdlib", false, "do not link the standard library")
	flag.Func("D", "define `NAME[=value]` (repeatable)", func(s string) error {
		name, value, _ := strings.Cut(s, "=")
		if name == "" {
			return errEmptyDefine
		}

		cfg.defines[name] = value

		return nil
	})
	flag.Parse()

	cfg.stdlib = !noStdlib
	cfg.inputFiles = flag.Args()

	switch {
	case len(cfg.inputFiles) == 0:
		return config{}, errNoInput
	case flag.NArg() == 0:
		return config{}, errNoInput
	case cfg.assembleOnly && cfg.preprocessOnly:
		return config{}, errBothModes
	}

	return cfg, nil
}
