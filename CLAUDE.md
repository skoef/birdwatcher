# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

birdwatcher is a health checker for [BIRD](https://bird.network.cz/)-anycasted services. It periodically runs shell commands to check service health and rewrites a BIRD configuration file (the generated `configfile`, default `/etc/bird/birdwatcher.conf`) to add or remove BGP prefixes based on service state. When the config changes, it reloads BIRD via a configurable command (`birdc configure`).

Note the two distinct paths: the program's own TOML config is read from `-config` (default `/etc/birdwatcher.conf`), while the BIRD prefix file it generates is the `configfile` key (default `/etc/bird/birdwatcher.conf`).

## Workflow

After changing any Go source code, always run the linter:

```bash
golangci-lint run
```

Fix all lint errors before considering the change complete.

Keep the Go version in `go.mod` (the `go` directive) in sync with the `go-version` used in the GitHub Actions workflows (`.github/workflows/go.yml` and `.github/workflows/golangci-lint.yml`). When bumping one, bump the others too.

## Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./birdwatcher/ -run TestName

# Run e2e tests (requires Docker)
go test -tags e2e -v -timeout 600s ./e2e/...

# Lint (requires golangci-lint installed)
golangci-lint run

# Check a config file
go run . -check-config -config /path/to/birdwatcher.conf

# Release (via goreleaser, triggered by git tags in CI)
goreleaser release --clean
```

### CLI flags (`main.go`)

- `-config` — path to the program's TOML config (default `/etc/birdwatcher.conf`)
- `-check-config` — validate config and exit (add `-debug` to also dump the parsed config as JSON)
- `-debug` — raise log level to debug
- `-systemd` — optimize for running under systemd (drops log timestamps, sends `sd_notify` readiness/status/stopping updates)
- `-version` — print version and exit

## Architecture

The program has a single main goroutine flow:

1. **`main.go`** — parses flags, reads config, optionally starts a Prometheus HTTP server, then creates a `HealthCheck` and calls `hc.Start(services, ready, sdStatus)` in a goroutine. It waits on the `ready` channel, optionally notifies systemd, then blocks until a signal (`SIGINT`/`SIGTERM`/`SIGQUIT`) is received before calling `hc.Stop()`. Under `-systemd` it forwards status strings from the `sdStatus` channel via `sd_notify`; the channel is always drained even without systemd so it never blocks the action loop.

2. **`birdwatcher/config.go`** — parses TOML config into `Config`, which holds a map of named `ServiceCheck` structs. `ReadConfig` validates the config and converts string prefixes to `net.IPNet`.

3. **`birdwatcher/servicecheck.go`** — each `ServiceCheck` runs in its own goroutine (via `s.Start`). On each tick it runs the configured `Command` (with `Timeout`), applies `Rise`/`Fail` counters to smooth transitions, and sends an `Action` on a shared channel when state changes (up→down or down→up).

4. **`birdwatcher/healthcheck.go`** — `HealthCheck.Start` fans out one goroutine per service and then loops on the `actions` channel. On each incoming `Action` it updates the in-memory `PrefixCollection`, then calls `applyConfig` which writes the new BIRD config and invokes the reload command.

5. **`birdwatcher/bird.go`** — writes the BIRD config by rendering `templates/functions.tpl` (an embedded Go template) into a temp file, then atomically renames it into place only if the content changed.

6. **`birdwatcher/prefixset.go`** — `PrefixCollection` is `map[string]*PrefixSet`, keyed by the BIRD function name (default `match_route`). Multiple services can share a function name or have distinct ones. `PrefixSet` stores `[]net.IPNet` and handles add/remove operations.

7. **`birdwatcher/action.go`** — simple struct carrying a `*ServiceCheck`, its new `ServiceState`, and the affected `[]net.IPNet`.

### Key design details

- Service checks run synchronously (no overlapping ticks): `performCheck` blocks the ticker loop, preventing check queues from building up.
- The first reload always happens even if the generated config is identical to the existing file (`reloadedBefore` flag), ensuring BIRD is in sync on startup.
- The `CompatBird213` flag removes `-> bool` return types from generated BIRD functions for compatibility with BIRD ≤ 2.13.
- The `functionname` config key allows multiple services to share or split a single `match_route`-style function in BIRD.
- Prometheus metrics are registered at package init via `promauto` — all metrics live in `birdwatcher/metrics.go`.

### Config format (TOML)

```toml
configfile = "/etc/bird/birdwatcher.conf"   # optional
reloadcommand = "/usr/sbin/birdc configure" # optional
compatbird213 = false                       # optional; strips "-> bool" return types for BIRD <= 2.13

[services]
  [services."my-service"]
  command = "/usr/bin/check.sh"
  prefixes = ["192.0.2.0/24", "2001:db8::/32"]
  interval = 1        # seconds
  timeout = "10s"
  fail = 1
  rise = 1
  functionname = "match_route"  # optional

[prometheus]
  enabled = false
  port = 9091
  path = "/metrics"
```
