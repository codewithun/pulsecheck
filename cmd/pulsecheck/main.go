package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	timeout := flag.Duration(
		"timeout",
		5*time.Second,
		"Maximum time allowed for each check (e.g. 2s, 500ms)",
	)

	tcpMode := flag.Bool(
		"tcp",
		false,
		"Check TCP ports instead of HTTP URLs",
	)

	jsonMode := flag.Bool(
		"json",
		false,
		"Output results as JSON",
	)

	flag.Parse()
	targets := flag.Args()

	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "Error: timeout must be greater than zero")
		os.Exit(1)
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr,
			"Usage: pulsecheck [-tcp] [-json] [-timeout duration] <target1> [target2] ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	results := make([]CheckResult, 0, len(targets))
	up := 0
	down := 0

	for _, target := range targets {
		var result CheckResult

		if *tcpMode {
			result = CheckTCP(target, *timeout)
		} else {
			result = CheckWithTimeout(target, *timeout)
		}

		result.LatencyMS = float64(result.Latency) / float64(time.Millisecond)
		results = append(results, result)

		if result.Status == "UP" {
			up++
		} else {
			down++
		}

		if !*jsonMode {
			fmt.Println(result.String())
		}
	}

	if *jsonMode {
		output := struct {
			Results []CheckResult `json:"results"`
			Summary struct {
				Up   int `json:"up"`
				Down int `json:"down"`
			} `json:"summary"`
		}{
			Results: results,
		}

		output.Summary.Up = up
		output.Summary.Down = down

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(output); err != nil {
			fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("\nSummary: %d UP, %d DOWN\n", up, down)
	}

	if down > 0 {
		os.Exit(1)
	}
}
