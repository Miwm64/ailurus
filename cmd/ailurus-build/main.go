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

func main() {
	cfg, err := parseConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(2)
	}

	linking := !cfg.assembleOnly && !cfg.preprocessOnly

	objects := make([]*isa.ObjectFile, 0)

	var errs []error

	for _, file := range cfg.inputFiles {
		src, err := io.ReadFile(file)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if filepath.Ext(file) == ".o" {
			if !linking {
				errs = append(errs, fmt.Errorf("%s: object file is only accepted when linking", file))

				continue
			}

			var obj isa.ObjectFile

			err := json.Unmarshal([]byte(src), &obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", file, err))

				continue
			}

			objects = append(objects, &obj)

			continue
		}

		res, lineMap, err := preprocessor.Preprocess(src, maps.Clone(cfg.defines))
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if cfg.preprocessOnly {
			_, _ = fmt.Println(res)

			continue
		}

		objectFile, err := assembler.Assemble(file, res, lineMap)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if cfg.assembleOnly {
			err := io.WriteJSON(strings.TrimSuffix(file, ".asm")+".o", objectFile)
			if err != nil {
				errs = append(errs, err)
			}

			continue
		}

		objects = append(objects, objectFile)
	}

	if cfg.stdlib && linking {
		res, lineMap, err := preprocessor.Preprocess(stdlib.Source, map[string]string{})
		if err != nil {
			errs = append(errs, err)
		} else if lib, err := assembler.Assemble("stdlib", res, lineMap); err != nil {
			errs = append(errs, err)
		} else {
			objects = append(objects, lib)
		}
	}

	if err := errors.Join(errs...); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	if !linking {
		return
	}

	image, listing, err := linker.Link(objects)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	if err := os.WriteFile(cfg.outputFile, image, 0o644); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	if cfg.listingFile != "" {
		err := os.WriteFile(cfg.listingFile, []byte(formatListing(listing)), 0o644)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)

			os.Exit(1)
		}
	}
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
