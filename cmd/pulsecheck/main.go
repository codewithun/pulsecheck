package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: pulsecheck <url>")
		os.Exit(1)
	}

	result := Check(os.Args[1])
	fmt.Println(result.String())

	if result.Status != "UP" {
		os.Exit(1)
	}
}
