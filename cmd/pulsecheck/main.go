package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pulsecheck <url>")
		os.Exit(1)
	}

	target := os.Args[1]

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(target)
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("DOWN %s | Error: %v | Latency: %s\n",
			target, err, latency.Round(time.Millisecond))
		os.Exit(1)
	}
	defer resp.Body.Close()

	status := "UP"
	exitCode := 0

	if resp.StatusCode >= 400 {
		status = "DOWN"
		exitCode = 1
	}

	fmt.Printf("%s %s | HTTP %d | Latency %s\n",
		status,
		target,
		resp.StatusCode,
		latency.Round(time.Millisecond),
	)

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
