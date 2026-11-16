package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Miwm64/ailurus/internal/assembler"
	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/isa"
	"github.com/Miwm64/ailurus/internal/linker"
	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	objects := make([]*isa.ObjectFile, 0)
	var errs []error

	for _, file := range cfg.inputFiles {
		test, err := io.ReadFile(file)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		res, lineMap, err := preprocessor.Preprocess(test, cfg.defines)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		objectFile, err := assembler.Assemble(file, res, lineMap)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		objects = append(objects, objectFile)
	}

	if err := errors.Join(errs...); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}

	_, _, err = linker.Link(objects)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}

}
