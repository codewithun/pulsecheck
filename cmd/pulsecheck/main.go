package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	timeout := flag.Duration(
		"timeout",
		5*time.Second,
		"Maximum time allowed for each HTTP check (e.g. 2s, 500ms)",
	)
	flag.Parse()

	targets := flag.Args()

	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "Error: timeout must be greater than zero")
		os.Exit(1)
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr,
			"Usage: pulsecheck [-timeout duration] <url1> [url2] ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	up := 0
	down := 0

	for _, target := range targets {
		result := CheckWithTimeout(target, *timeout)
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
