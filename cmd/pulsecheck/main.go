package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type CheckSummary struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

type CheckReport struct {
	Results []CheckResult `json:"results"`
	Summary CheckSummary  `json:"summary"`
}

func main() {
	timeout := flag.Duration(
		"timeout", 5*time.Second,
		"Maximum time allowed for each check",
	)

	tcpMode := flag.Bool(
		"tcp", false,
		"Check TCP ports instead of HTTP URLs",
	)

	jsonMode := flag.Bool(
		"json", false,
		"Output results as JSON",
	)

	interval := flag.Duration(
		"interval", 0,
		"Repeat checks at the specified interval (e.g. 10s); 0 disables repetition",
	)

	flag.Parse()
	targets := flag.Args()

	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "Error: timeout must be greater than zero")
		os.Exit(1)
	}

	if *interval < 0 {
		fmt.Fprintln(os.Stderr, "Error: interval cannot be negative")
		os.Exit(1)
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr,
			"Usage: pulsecheck [-tcp] [-json] [-timeout duration] [-interval duration] <target1> [target2] ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	run := func() bool {
		report := CheckTargets(targets, *timeout, *tcpMode)

		if *jsonMode {
			encoder := json.NewEncoder(os.Stdout)

			// In periodic mode, each line is a separate JSON report.
			// This format is commonly called JSON Lines.
			if *interval == 0 {
				encoder.SetIndent("", "  ")
			}

			if err := encoder.Encode(report); err != nil {
				fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n[%s]\n", time.Now().Format(time.RFC3339))

			for _, result := range report.Results {
				fmt.Println(result.String())
			}

			fmt.Printf("Summary: %d UP, %d DOWN\n",
				report.Summary.Up, report.Summary.Down)
		}

		return report.Summary.Down == 0
	}

	if *interval == 0 {
		if !run() {
			os.Exit(1)
		}
		return
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	fmt.Fprintf(os.Stderr,
		"Starting periodic checks every %s. Press Ctrl+C to stop.\n",
		*interval)

	run()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\nPulseCheck stopped.")
			return
		case <-ticker.C:
			run()
		}
	}
}

func CheckTargets(
	targets []string,
	timeout time.Duration,
	tcpMode bool,
) CheckReport {
	report := CheckReport{
		Results: make([]CheckResult, 0, len(targets)),
	}

	for _, target := range targets {
		var result CheckResult

		if tcpMode {
			result = CheckTCP(target, timeout)
		} else {
			result = CheckWithTimeout(target, timeout)
		}

		result.LatencyMS = float64(result.Latency) / float64(time.Millisecond)
		report.Results = append(report.Results, result)

		if result.Status == "UP" {
			report.Summary.Up++
		} else {
			report.Summary.Down++
		}
	}

	return report
}
