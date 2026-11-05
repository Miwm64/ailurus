package util

import "fmt"

func PrintMap(m map[string]string) {
	for k, v := range m {
		fmt.Printf("%s: %s\n", k, v)
	}
}

func PrintSlice(str []string) {
	for i, v := range str {
		fmt.Printf("%d: %s\n", i, v)
	}
}
