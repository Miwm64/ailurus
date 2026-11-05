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

		#ifndef hi
		#ifndef hi23231
		123
		#endif
		#else
		321
		#endif

		#macro aboba, 3 {
			add r2, 41
			add r2, %0
			%1, %2
		}
		sub 123, hi2
		aboba 1, "123", hi
		`
	res, lineMap, err := preprocessor.Preprocess(test, make(map[string]string))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res)
	fmt.Println(lineMap)
}
