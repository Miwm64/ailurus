package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/assembler"
	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/isa"
	"github.com/Miwm64/ailurus/internal/linker"
	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	testFile := "obj.asm"
	test, err := io.ReadFile("programs/" + testFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	res, lineMap, err := preprocessor.Preprocess(test, make(map[string]string))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = io.WriteFile("programs/obj_res.asm", res)
	if err != nil {
		fmt.Println(err)
	}

	objectFile, err := assembler.Assemble(testFile, res, lineMap)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = io.WriteJSON("programs/obj.o", objectFile)
	if err != nil {
		fmt.Println(err)
	}

	bin, entries, err := linker.Link([]*isa.ObjectFile{objectFile})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", bin)
	fmt.Printf("%s\n", entries)

}
