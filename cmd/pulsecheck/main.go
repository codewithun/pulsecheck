package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
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

	concurrency := flag.Int(
		"concurrency", 10,
		"Maximum number of targets to check simultaneously",
	)

	retries := flag.Int(
		"retries", 0,
		"Number of retries when a check fails",
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

	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "Error: concurrency must be at least 1")
		os.Exit(1)
	}

	if *retries < 0 {
		fmt.Fprintln(os.Stderr, "Error: retries cannot be negative")
		os.Exit(1)
	}

	if len(targets) == 0 {
		fmt.Fprintln(
			os.Stderr,
			"Usage: pulsecheck [-tcp] [-json] [-timeout duration] [-interval duration] [-concurrency n] [-retries n] <target1> [target2] ...",
		)
		flag.PrintDefaults()
		os.Exit(1)
	}

	run := func() bool {
		report := CheckTargets(
			targets,
			*timeout,
			*tcpMode,
			*concurrency,
			*retries,
		)

		if *jsonMode {
			encoder := json.NewEncoder(os.Stdout)

			// In periodic mode, each line is a separate JSON report.
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

			fmt.Printf(
				"Summary: %d UP, %d DOWN\n",
				report.Summary.Up,
				report.Summary.Down,
			)
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

	fmt.Fprintf(
		os.Stderr,
		"Starting periodic checks every %s. Press Ctrl+C to stop.\n",
		*interval,
	)

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
	concurrency int,
	retries int,
) CheckReport {
	report := CheckReport{
		Results: make([]CheckResult, len(targets)),
	}

	if len(targets) == 0 {
		return report
	}

	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(targets) {
		concurrency = len(targets)
	}

	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, target := range targets {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(i int, target string) {
			defer wg.Done()
			defer func() {
				<-semaphore
			}()

			var result CheckResult

			// retries=2 means up to 3 total attempts.
			for attempt := 0; attempt <= retries; attempt++ {
				if tcpMode {
					result = CheckTCP(target, timeout)
				} else {
					result = CheckWithTimeout(target, timeout)
				}

				// Retry only failed checks, not invalid targets.
				if result.Status != "DOWN" {
					break
				}
			}

			result.LatencyMS = float64(result.Latency) /
				float64(time.Millisecond)

			report.Results[i] = result
		}(i, target)
	}

	wg.Wait()

	for _, result := range report.Results {
		if result.Status == "UP" {
			report.Summary.Up++
		} else {
			report.Summary.Down++
		}
	}

	return report
}
