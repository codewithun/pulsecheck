# PulseCheck

A lightweight HTTP health checker written in Go.

PulseCheck checks the availability of websites and HTTP endpoints, measures response latency, detects unhealthy HTTP status codes, and supports configurable request timeouts.

## Features

- Check one or multiple URLs in a single command
- Report HTTP status codes and response latency
- Detect unhealthy HTTP responses (4xx and 5xx)
- Validate target URLs
- Configure request timeout
- Summarize successful and failed checks
- Automated unit tests for HTTP responses and timeout behavior

## Requirements

- Go (use a version compatible with the module configuration)

## Installation

Clone the repository:

```bash
git clone https://github.com/codewithun/pulsecheck.git
cd pulsecheck
```

## Usage

Check a single URL:

```bash
go run ./cmd/pulsecheck https://example.com
```

Check multiple URLs:

```bash
go run ./cmd/pulsecheck https://google.com https://github.com
```

Configure the timeout:

```bash
go run ./cmd/pulsecheck -timeout 3s https://example.com
```

Use a shorter timeout:

```bash
go run ./cmd/pulsecheck -timeout 500ms https://example.com
```

## Example Output

```text
UP https://example.com | HTTP 200 | Latency 180ms

Summary: 1 UP, 0 DOWN
```

_Latency in this example is illustrative._

## Health Check Behavior

- `UP`: The endpoint returns an HTTP status code below 400.
- `DOWN`: The endpoint returns an HTTP status code of 400 or higher, or the request fails.
- `INVALID`: The target URL or timeout configuration is invalid.

The process exits with a non-zero status when an argument error occurs or at least one check fails.

## Development

Run tests:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

## Project Status

PulseCheck is an ongoing learning project focused on Go, HTTP networking, error handling, and automated testing.

## License

No license has been added yet. Licensing terms will be specified in a future update.
