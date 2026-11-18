package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/Miwm64/ailurus/internal/assembler"
	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/isa"
	"github.com/Miwm64/ailurus/internal/linker"
	"github.com/Miwm64/ailurus/internal/preprocessor"
	"github.com/Miwm64/ailurus/internal/stdlib"
)

var errObjectNotLinked = errors.New("object file is only accepted when linking")

func main() {
	cfg, err := parseConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(2)
	}

	switch {
	case cfg.preprocessOnly:
		err = forEachSource(cfg.inputFiles, func(_, src string) error {
			return printPreprocessed(src, cfg.defines)
		})
	case cfg.assembleOnly:
		err = forEachSource(cfg.inputFiles, func(file, src string) error {
			return writeObject(file, src, cfg.defines)
		})
	default:
		err = build(cfg.inputFiles, cfg.defines, cfg.stdlib, cfg.outputFile, cfg.listingFile)
	}

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func forEachSource(files []string, do func(file, src string) error) error {
	var errs []error

	for _, file := range files {
		src, err := io.ReadFile(file)
		if err == nil && filepath.Ext(file) == ".o" {
			err = fmt.Errorf("%s: %w", file, errObjectNotLinked)
		}

		if err == nil {
			err = do(file, src)
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func printPreprocessed(src string, defines map[string]string) error {
	res, _, err := preprocessor.Preprocess(src, maps.Clone(defines))
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, res)

	return nil
}

func writeObject(file, src string, defines map[string]string) error {
	obj, err := assemble(file, src, defines)
	if err != nil {
		return err
	}

	return io.WriteJSON(strings.TrimSuffix(file, ".asm")+".o", obj)
}

func build(files []string, defines map[string]string, withStdlib bool, output, listingFile string) error {
	objects := make([]*isa.ObjectFile, 0, len(files)+1)

	var errs []error

	for _, file := range files {
		obj, err := loadObject(file, defines)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		objects = append(objects, obj)
	}

	if withStdlib {
		lib, err := assemble("stdlib", stdlib.Source, map[string]string{})
		if err != nil {
			errs = append(errs, err)
		} else {
			objects = append(objects, lib)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	image, listing, err := linker.Link(objects)
	if err != nil {
		return err
	}

	if err := os.WriteFile(output, image, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if listingFile == "" {
		return nil
	}

	if err := os.WriteFile(listingFile, []byte(formatListing(listing)), 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func loadObject(file string, defines map[string]string) (*isa.ObjectFile, error) {
	src, err := io.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if filepath.Ext(file) != ".o" {
		return assemble(file, src, defines)
	}

	var obj isa.ObjectFile
	if err := json.Unmarshal([]byte(src), &obj); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}

	return &obj, nil
}

func assemble(name, src string, defines map[string]string) (*isa.ObjectFile, error) {
	res, lineMap, err := preprocessor.Preprocess(src, maps.Clone(defines))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	obj, err := assembler.Assemble(name, res, lineMap)
	return obj, fmt.Errorf("%s: %w", name, err)
}

func formatListing(entries []linker.ListingEntry) string {
	var b strings.Builder

	for _, e := range entries {
		line := fmt.Sprintf("%08X - %08X", e.Address, e.Word)
		if e.Mnemonic != "" {
			line += " - " + e.Mnemonic
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}
