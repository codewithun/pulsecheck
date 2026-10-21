package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: pulsecheck <url>")
		os.Exit(1)
	}

	target := strings.TrimSpace(os.Args[1])

	parsedURL, err := url.ParseRequestURI(target)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		fmt.Printf("Invalid URL: %s\n", target)
		fmt.Println("URL must start with http:// or https://")
		os.Exit(1)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(parsedURL.String())
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("DOWN %s | Error: %v | Latency: %s\n",
			target, err, latency.Round(time.Millisecond))
		os.Exit(1)
	}
	defer resp.Body.Close()

	status := "UP"
	if resp.StatusCode >= 400 {
		status = "DOWN"
	}

	fmt.Printf("%s %s | HTTP %d | Latency %s\n",
		status,
		target,
		resp.StatusCode,
		latency.Round(time.Millisecond),
	)

	if status == "DOWN" {
		os.Exit(1)
	}
}
