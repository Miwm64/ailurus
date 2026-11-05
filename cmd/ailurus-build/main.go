package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	test :=
		`
		#define hi
		#define hi2 hi
		123

		`
	res := preprocessor.Preprocess(test, make(map[string]string))
	fmt.Println(res)
}
