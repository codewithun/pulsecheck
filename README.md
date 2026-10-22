# PulseCheck

A lightweight command-line tool written in Go for checking HTTP endpoint availability and TCP port connectivity.

PulseCheck helps developers and system administrators quickly verify whether a web endpoint responds successfully or a TCP service accepts connections.

## Features

- **HTTP health checks** — Check one or multiple HTTP/HTTPS endpoints.
- **TCP port checks** — Verify connectivity to TCP services using host and port.
- **Configurable timeout** — Set the maximum time allowed for each check.
- **Latency measurement** — Display how long each check takes.
- **Input validation** — Reject invalid URLs, addresses, and port numbers.
- **Summary report** — Count successful and failed checks.
- **Automation-friendly exit codes** — Return a non-zero exit code when a check fails.

## Requirements

- Go 1.23 or later recommended.
- Network access to the targets being checked.

## Installation

Clone the repository:

```bash
git clone https://github.com/codewithun/pulsecheck.git
cd pulsecheck
```

Run the test suite:

```bash
go test ./...
```

Run PulseCheck directly:

```bash
go run ./cmd/pulsecheck https://google.com
```

## Usage

### HTTP checks

Check a single endpoint:

```bash
go run ./cmd/pulsecheck https://google.com
```

Check multiple endpoints:

```bash
go run ./cmd/pulsecheck https://google.com https://github.com
```

Set a custom timeout:

```bash
go run ./cmd/pulsecheck -timeout 2s https://google.com
```

The default timeout is 5 seconds per check.

### TCP port checks

Check a single TCP service:

```bash
go run ./cmd/pulsecheck -tcp localhost:5432
```

Check multiple TCP ports:

```bash
go run ./cmd/pulsecheck -tcp localhost:22 localhost:5432
```

Set a custom timeout:

```bash
go run ./cmd/pulsecheck -tcp -timeout 2s localhost:22 localhost:5432
```

TCP targets must use the `host:port` format, and port numbers must be between 1 and 65535.

## Example Output

HTTP check:

```text
UP https://google.com | HTTP 200 | Latency 120ms

Summary: 1 UP, 0 DOWN
```

TCP check:

```text
UP localhost:5432 | TCP connection successful | Latency: 1ms

Summary: 1 UP, 0 DOWN
```

Output and latency values depend on the target and network conditions.

## Status Definitions

| Status    | Meaning                                                                                                         |
| --------- | --------------------------------------------------------------------------------------------------------------- |
| `UP`      | The HTTP request succeeded according to PulseCheck's status rules, or the TCP connection was established.       |
| `DOWN`    | The request failed, the HTTP response status was 400 or higher, or the TCP connection could not be established. |
| `INVALID` | The supplied URL, address, port, or timeout is invalid.                                                         |

A `DOWN` result does not necessarily mean the entire server is offline. The service might be unavailable, the port may be closed, or a firewall may be blocking the connection.

## Exit Codes

- `0` — All targets passed their checks.
- `1` — At least one target did not pass, or the command received invalid input.

These exit codes make PulseCheck suitable for use in shell scripts and automation workflows.

## Testing and Code Quality

Run unit tests:

```bash
go test ./...
```

Check code formatting:

```bash
gofmt -w cmd/pulsecheck/*.go
```

Run static analysis:

```bash
go vet ./...
```

## Project Structure

```text
pulsecheck/
├── cmd/
│   └── pulsecheck/
│       ├── main.go
│       ├── checker.go
│       └── checker_test.go
├── go.mod
└── README.md
```

## Roadmap

- [x] HTTP endpoint checks
- [x] Configurable HTTP timeout
- [x] HTTP status and latency reporting
- [x] HTTP redirect handling
- [x] TCP port checks
- [x] Unit tests
- [ ] Continuous integration with GitHub Actions
- [ ] JSON output for automation
- [ ] Periodic health checks

## License

A license has not yet been specified. Add a `LICENSE` file before distributing the project under an open-source license.
