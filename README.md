# birdwatcher

[![Go Report Card](https://goreportcard.com/badge/github.com/skoef/birdwatcher)](https://goreportcard.com/report/github.com/skoef/birdwatcher)

> healthchecker for [BIRD](https://bird.network.cz/)-anycasted services

:warning: **NOTE**: this version of birdwatcher is designed to work with BIRD version 2 and up. If you want to use birdwatcher with BIRD 1.x, refer to the `bird1` branch.

This project is heavily influenced by [anycast-healthchecker](https://github.com/unixsurfer/anycast_healthchecker). If you want to know more about use cases of birdwatcher, please read their excellent documentation about anycasted services and how a healthchecker can contribute to a more stable service availability.

In a nutshell: birdwatcher periodically checks a specific service and tells BIRD which prefixes to announce or to withdraw when the service appears to be up or down respectively.

## Why birdwatcher

When I found out about anycast-healthchecker on [HaproxyConf 2019](https://www.haproxyconf.com/), I figured this would solve the missing link between BIRD and whatever anycasted services I had running (mostly haproxy though). Currently however in anycast-healthchecker, it is not possible to specify multiple prefixes to a service. Some machines in these kind of setups are announcing _many_ `/32` and `/128` prefixes and I ended up specifying so many services (one per prefix) that python crashed giving me a `too many open files` error. At first I tried to patch anycast-healthchecker but ended up writing something similar, hence birdwatcher.

It is written in Go because I like it and running multiple threads is easy.

## Upgrading

### Upgrading from 1.0.0-beta2 to 1.0.0-beta3

The notation of `timeout` for services changed from int (for number of seconds) to the format `time.ParseDuration` uses, such as "300ms", "-1.5h" or "2h45m".

For example: `10` should become `"10s"` in your config files.

## Example usage

This simple example configures a single service, runs `haproxy_check.sh` every second and manages one prefix based on the exit code of the script:

```toml
[services]
  [services."foo"]
  command = "/usr/bin/haproxy_check.sh"
  prefixes = ["192.168.0.0/24"]

```

Sample output in `/etc/bird/birdwatcher.conf` if `haproxy_check.sh` checks out would be:

```text
# DO NOT EDIT MANUALLY
function match_route() -> bool
{
	return net ~ [
		192.168.0.0/24
	];
}
```

As soon as birdwatcher finds out haproxy is down, it will change the content in `/etc/bird/birdwatcher.conf` to:

```text
# DO NOT EDIT MANUALLY
function match_route() -> bool
{
	return false;
}
```

and reconfigures BIRD by given `reloadcommand`. Obviously, if you have multiple services being checked by birdwatcher, only the prefixes of that particular service would be removed from the list in `match_route`.

### Multiple prefixes (IPv4 + IPv6)

When announcing both IPv4 and IPv6 prefixes, you **must** split them into separate services with different function names. BIRD does not allow mixed IPv4/IPv6 prefixes in a single BGP protocol block, so birdwatcher generates separate functions for each address family:

```toml
[services]
  # IPv4-only service
  [services."haproxy-4"]
  command = "/usr/bin/haproxy_check.sh"
  prefixes = ["10.0.0.1/32"]

  # IPv6-only service (note the different functionname!)
  [services."haproxy-6"]
  command = "/usr/bin/haproxy_check.sh"
  functionname = "match_route_ipv6"
  prefixes = ["fd00::1/128"]
```

This generates two separate functions in `/etc/bird/birdwatcher.conf`:

```
# DO NOT EDIT MANUALLY
function match_route() -> bool
{
	return net ~ [
		10.0.0.1/32
	];
}
function match_route_ipv6() -> bool
{
	return net ~ [
		fd00::1/128
	];
}
```

In BIRD, use the corresponding function in each address family's protocol block:

```
include "/etc/bird/birdwatcher.conf";

protocol bgp my_bgp_v4 {
  local as 12345;
  neighbor 1.2.3.4 as 23435;
  ...
  ipv4 {
    ...
    export where match_route();
  }
}

protocol bgp my_bgp_v6 {
  local as 12345;
  neighbor 1.2.3.4 as 23435;
  ...
  ipv6 {
    ...
    export where match_route_ipv6();
  }
}
```

**Important:**
- Each service must contain **only** IPv4 or **only** IPv6 prefixes
- Each service needs a **unique `functionname`** (defaults to `match_route`)
- The generated function name must match the one referenced in your BIRD `export where` filter
- Place `include "/etc/bird/birdwatcher.conf";` at the **top** of your bird.conf — before any protocol definitions that reference the functions
## Command-line flags

| flag            | description                                                                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `-config`       | Path to birdwatcher's own TOML config file. Defaults to **/etc/birdwatcher.conf**.                                                                      |
| `-check-config` | Validate the config file and exit. Combine with `-debug` to also dump the parsed config as JSON.                                                        |
| `-debug`        | Increase the log level to debug.                                                                                                                        |
| `-systemd`      | Optimize behavior for running under systemd: drop log timestamps (journald adds its own) and send `sd_notify` readiness, status and stopping updates.   |
| `-version`      | Print the version and exit.                                                                                                                             |

Note the two distinct paths: the `-config` flag points to birdwatcher's own TOML
config (default **/etc/birdwatcher.conf**), while the `configfile` key _inside_
that config is the BIRD prefix file birdwatcher generates (default
**/etc/bird/birdwatcher.conf**).

For example, to validate a config file without starting birdwatcher:

```bash
birdwatcher -check-config -config /etc/birdwatcher.conf
```

## Configuration

## **global**

Configuration section for global options.

| key           | description                                                                                                                                     |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| configfile    | Path to configuration file that will be generated and should be included in the BIRD configuration. Defaults to **/etc/bird/birdwatcher.conf**. |
| reloadcommand | Command to invoke to signal BIRD the configuration should be reloaded. Defaults to **/usr/sbin/birdc configure**.                               |
| compatbird213 | To use birdwatcher with BIRD 2.13 or earlier, enable this flag. It will remove the function return types from the output                        |

## **[services]**

Each service under this section can have the following settings:

| key          | description                                                                                                                                                                                                                              |
| ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| command      | Command that will be periodically run to check if the service should be considered up or down. The result is based on the exit code: a non-zero exit codes makes birdwatcher decide the service is down, otherwise it's up. **Required** |
| functionname | Specify the name of the function birdwatcher will generate. You can use this function name to use in your protocol export filter in BIRD. Defaults to **match_route**.                                                                   |
| interval     | The interval in seconds at which birdwatcher will check the service. Defaults to **1**                                                                                                                                                   |
| timeout      | Time in which the check command should complete. Afterwards it will be handled as if the check command failed. Defaults to **10s**, format following that of [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration).              |
| fail         | The amount of times the check command should fail before the service is considered to be down. Defaults to **1**                                                                                                                         |
| rise         | The amount of times the check command should succeed before the service is considered to be up. Defaults to **1**                                                                                                                        |
| prefixes     | Array of prefixes, mixed IPv4 and IPv6. At least 1 prefix is **required** per service                                                                                                                                                    |

## **[prometheus]**

Configuration for the prometheus exporter

| key     | description                                                                  |
| ------- | ---------------------------------------------------------------------------- |
| enabled | Boolean whether you want to export prometheus metrics. Defaults to **false** |
| port    | Port to export prometheus metrics on. Defaults to **9091**                   |
| path    | Path to the prometheus metrics. Defaults to **/metrics**                     |

## Testing

The regular unit tests run with:

```bash
go test ./...
```

### End-to-end tests

The `e2e/` package contains an end-to-end test that exercises the full loop: it
spins up a real BIRD instance in a container (via
[testcontainers-go](https://golang.testcontainers.org/)), runs a real
`birdwatcher` binary against it, and verifies through `birdc` that prefixes are
announced and withdrawn as the service health changes.

Because it launches containers, it is guarded behind the `e2e` build tag and is
**not** part of `go test ./...`. To run it you need a working Docker daemon
(Docker Desktop, [Colima](https://github.com/abiosoft/colima), or a native
Docker Engine):

```bash
go test -tags e2e -v -timeout 600s ./e2e/...
```

The test builds the `birdwatcher` binary itself, so no separate build step is
required. It automatically disables the testcontainers Ryuk reaper
(`TESTCONTAINERS_RYUK_DISABLED=true`), which works around issues with
non-standard Docker socket paths; containers are still cleaned up when the test
finishes.

On macOS with Colima, note that only your home directory is mounted into the
Docker VM by default. The test accounts for this by creating its temporary
working directory under your home directory rather than `/tmp`.

This is the same command the CI `e2e` job runs (see
`.github/workflows/go.yml`).
