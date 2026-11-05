package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/io"
	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	test, err := io.ReadFile("test.asm")
	if err != nil {
		fmt.Println(err)
		return
	}
	res, _, err := preprocessor.Preprocess(test, make(map[string]string))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = io.WriteFile("test_res.asm", res)
	if err != nil {
		fmt.Println(err)
	}
}
