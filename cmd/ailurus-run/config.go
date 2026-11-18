package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/Miwm64/ailurus/internal/vm"
)

const (
	defaultMemorySize = 4096
	defaultTickLimit  = 100_000

	journalInstr = "instr"
	journalTick  = "tick"

	inputEntryLen = 2
	listingFields = 3 // address - HEXCODE - mnemonic
)

var (
	errNoImage         = errors.New("no image file")
	errManyImages      = errors.New("exactly one image file expected")
	errBadInput        = errors.New("bad input entry")
	errBadJournalMode  = errors.New("journal mode must be instr or tick")
	errBadJournalRange = errors.New("journal: from_tick > to_tick")
	errBadListing      = errors.New("bad listing line")
)

type config struct {
	imageFile   string
	listingFile string
	configFile  string
	journalFile string
}

type machineConfig struct {
	MemorySize uint32        `yaml:"memory_size"`
	Cache      bool          `yaml:"cache"`
	TickLimit  uint64        `yaml:"tick_limit"`
	Input      [][]any       `yaml:"input"`
	Journal    journalConfig `yaml:"journal"`
}

type journalConfig struct {
	Mode     string `yaml:"mode"`
	FromTick uint64 `yaml:"from_tick"`
	ToTick   uint64 `yaml:"to_tick"`
}

func parseConfig() (config, error) {
	var cfg config

	flag.StringVar(&cfg.listingFile, "l", "", "debug listing `file` from ailurus-build")
	flag.StringVar(&cfg.configFile, "c", "", "simulator config `file` (YAML)")
	flag.StringVar(&cfg.journalFile, "j", "", "journal output `file`")
	flag.Parse()

	switch flag.NArg() {
	case 0:
		return config{}, errNoImage
	case 1:
		cfg.imageFile = flag.Arg(0)
	default:
		return config{}, errManyImages
	}

	return cfg, nil
}

func parseMachineConfig(src string) (vm.Config, journalConfig, error) {
	parsed := machineConfig{
		MemorySize: defaultMemorySize,
		Cache:      true,
		TickLimit:  defaultTickLimit,
		Journal:    journalConfig{Mode: journalInstr, ToTick: math.MaxUint64},
	}

	dec := yaml.NewDecoder(strings.NewReader(src))
	dec.KnownFields(true)

	if err := dec.Decode(&parsed); err != nil && !errors.Is(err, io.EOF) {
		return vm.Config{}, journalConfig{}, fmt.Errorf("config: %w", err)
	}

	input, err := parseInput(parsed.Input)
	if err != nil {
		return vm.Config{}, journalConfig{}, err
	}

	switch {
	case parsed.Journal.Mode != journalInstr && parsed.Journal.Mode != journalTick:
		return vm.Config{}, journalConfig{}, errBadJournalMode
	case parsed.Journal.FromTick > parsed.Journal.ToTick:
		return vm.Config{}, journalConfig{}, errBadJournalRange
	}

	return vm.Config{
		MemorySize: parsed.MemorySize,
		Cache:      parsed.Cache,
		TickLimit:  parsed.TickLimit,
		Input:      input,
	}, parsed.Journal, nil
}

func parseInput(raw [][]any) ([]vm.InputEvent, error) {
	events := make([]vm.InputEvent, 0, len(raw))
	for i, pair := range raw {
		if len(pair) != inputEntryLen {
			return nil, fmt.Errorf("%w %d: want [tick, \"c\"]", errBadInput, i)
		}

		tick, okTick := pair[0].(int)

		char, okChar := pair[1].(string)
		if !okTick || !okChar || tick < 0 || len(char) != 1 {
			return nil, fmt.Errorf("%w %d: want [tick, \"c\"]", errBadInput, i)
		}

		if i > 0 && uint64(tick) <= events[i-1].Tick {
			return nil, fmt.Errorf("%w %d: ticks must be ascending", errBadInput, i)
		}

		events = append(events, vm.InputEvent{Tick: uint64(tick), Char: char[0]})
	}

	return events, nil
}

func formatJournal(records []vm.Record, jc journalConfig, source map[uint32]string) string {
	var b strings.Builder

	for _, r := range records {
		if r.Tick < jc.FromTick || r.Tick > jc.ToTick {
			continue
		}

		if jc.Mode == journalTick {
			b.WriteString(r.TickLine() + "\n")
		}

		if r.InstrDone {
			b.WriteString(r.InstructionLine(source[r.InstrAddr]) + "\n")
		}
	}

	return b.String()
}

func parseListing(src string) (map[uint32]string, error) {
	source := map[uint32]string{}

	for i, line := range strings.Split(src, "\n") {
		fields := strings.SplitN(line, " - ", listingFields)
		if len(fields) < listingFields {
			continue
		}

		addr, err := strconv.ParseUint(fields[0], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("%w %d: %w", errBadListing, i+1, err)
		}

		source[uint32(addr)] = fields[2]
	}

	return source, nil
}

// run summary.
func summary(ticks uint64, runErr error) string {
	switch {
	case runErr == nil:
		return fmt.Sprintf("halted after %d ticks", ticks)
	case errors.Is(runErr, vm.ErrTickLimit):
		return fmt.Sprintf("tick limit reached after %d ticks", ticks)
	default:
		return fmt.Sprintf("fault after %d ticks: %v", ticks, runErr)
	}
}
