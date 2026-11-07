package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/assembler"
	"github.com/Miwm64/ailurus/internal/io"
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
	err = io.WriteFile("programs/"+testFile, res)
	if err != nil {
		fmt.Println(err)
	}

	_, err = assembler.Assemble(testFile, res, lineMap)
	if err != nil {
		fmt.Println(err)
	}
}
