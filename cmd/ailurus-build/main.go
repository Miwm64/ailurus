package main

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/preprocessor"
)

func main() {
	test :=
		`
		#define hi
		#define hi2  hi
		#define hi2  hi
		#define r1  hi
		123

		`
	err, res := preprocessor.Preprocess(test, make(map[string]string))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res)
}
