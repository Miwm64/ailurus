package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/preprocess"
)

func main() {
	test := "hi!"
	res := preprocess.Preprocess(test)
	fmt.Println(res)
}
