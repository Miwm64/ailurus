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

	configSrc := ""
	if cfg.configFile != "" {
		configSrc, err = io.ReadFile(cfg.configFile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)

			os.Exit(1)
		}
	}

	machineCfg, journalCfg, err := parseMachineConfig(configSrc)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(2)
	}

	source := map[uint32]string{}

	if cfg.listingFile != "" {
		listingSrc, err := io.ReadFile(cfg.listingFile)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)

			os.Exit(1)
		}

		source, err = parseListing(listingSrc)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)

			os.Exit(1)
		}
	}

	image, err := io.ReadBytes(cfg.imageFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	machine, err := vm.New(image, machineCfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	runErr := machine.Run()

	if cfg.journalFile != "" {
		journal := formatJournal(machine.Journal(), journalCfg, source)

		err := io.WriteFile(cfg.journalFile, journal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)

			os.Exit(1)
		}
	}

	_, _ = fmt.Fprintln(os.Stderr, summary(machine.Ticks(), runErr))
	_, _ = fmt.Fprint(os.Stderr, statsText(machine.Stats()))
	_, _ = os.Stdout.Write(machine.Output())

	if runErr != nil {
		os.Exit(1)
	}
}

func statsText(stats vm.Stats) string {
	text := fmt.Sprintf("memory wait: %d ticks\n", stats.MemWait)

	if c := stats.Cache; c != nil {
		text += fmt.Sprintf("cache: %d hits, %d misses", c.Hits, c.Misses)
		if reads := c.Hits + c.Misses; reads > 0 {
			text += fmt.Sprintf(" (%.1f%% hit rate)", 100*float64(c.Hits)/float64(reads))
		}

		text += fmt.Sprintf(", %d evictions, %d writes, %d bypasses\n", c.Evictions, c.Writes, c.Bypasses)
	}

	return text
}
