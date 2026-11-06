package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/assembler"
	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	test, err := io.ReadFile("programs/obj.asm")
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

	_, err = assembler.Assemble(res, lineMap)
	if err != nil {
		fmt.Println(err)
	}
}
