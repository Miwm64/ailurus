package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	test := "hi!"
	res := preprocessor.Preprocess(test)
	fmt.Println(res)
}
