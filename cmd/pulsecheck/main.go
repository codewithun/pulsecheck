package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pulsecheck <url1> [url2] [url3] ...")
		os.Exit(1)
	}

	up := 0
	down := 0

	for _, target := range os.Args[1:] {
		result := Check(target)
		fmt.Println(result.String())

		if result.Status == "UP" {
			up++
		} else {
			down++
		}
	}

	fmt.Printf("\nSummary: %d UP, %d DOWN\n", up, down)

	if down > 0 {
		os.Exit(1)
	}
}
