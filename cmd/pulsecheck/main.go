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
		fmt.Printf("DOWN %s: %v\n", target, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf(
		"UP %s | HTTP %d | Latency %s\n",
		target,
		resp.StatusCode,
		latency.Round(time.Millisecond),
	)
}
