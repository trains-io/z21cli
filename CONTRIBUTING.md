# Contributing

Thanks for contributing to [github.com/trains-io/z21cli](https://github.com/trains-io/z21cli).

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Build and test |
| Docker | recent | Run the [z21-sim](https://github.com/trains-io/z21-sim) test server locally |

## Build

From the repository root:

```bash
make build    # bin/z21
make clean    # remove bin/
```

Equivalent manual command:

```bash
go build -o bin/z21 .
```

Install from source:

```bash
go install .
```

## Test

Run all unit tests:

```bash
make test
```

Or directly:

```bash
go test ./... -count=1
```

Filter by package or test name:

```bash
go test ./cmd/... -run TestContext -v -count=1
go test ./internal/contextstore/... -count=1
```

Use `-count=1` while iterating to disable the test cache.

CLI tests are unit tests only; they do not require Docker or a running simulator.

## Run against the z21 simulator

Manual testing uses [z21-sim](https://github.com/trains-io/z21-sim), a headless ZIMO `Z21Posix` process published as `ghcr.io/trains-io/z21-sim:latest`. It listens on UDP port **21105**.

### 1. Start the simulator

In one terminal:

```bash
docker run --rm -d --name z21-sim -p 21105:21105/udp ghcr.io/trains-io/z21-sim:latest
```

If port 21105 is already in use, map a different host port:

```bash
docker run --rm -d --name z21-sim -p 21107:21105/udp ghcr.io/trains-io/z21-sim:latest
```

When using a custom host port, pass `--port` to CLI commands (see below).

View logs:

```bash
docker logs -f z21-sim
```

Stop the simulator with:

```bash
docker stop z21-sim
```

### 2. Build the CLI

In a second terminal:

```bash
make build
```

### 3. Ping without a saved context

Quick connectivity check (no context file needed):

```bash
./bin/z21 ping --host 127.0.0.1
```

With a custom host port:

```bash
./bin/z21 ping --host 127.0.0.1 --port 21107
```

Expected output:

```bash
pong from 127.0.0.1:21105 (hw=0x00000201 fw=0x00000143)
```

### 4. Use a named context (recommended)

For repeated work against the same command station, save a context in `~/.z21_contexts.json`:

```bash
./bin/z21 context add sim --host 127.0.0.1 --use
./bin/z21 ping
./bin/z21 status
./bin/z21 context show
```

Switch contexts with `context use` (alias `ctx use`).

### 5. Try subscriptions and monitoring

Subscribe to system updates, then watch live broadcast events:

```bash
./bin/z21 sub add SYSTEM_UPDATES
./bin/z21 monitor
```

Press `Ctrl+C` to stop monitoring.

### 6. Debug UDP traffic

Add `--trace` to any command to hexdump Z21 requests and responses on stderr:

```bash
./bin/z21 --trace ping --host 127.0.0.1
```

## CI

GitHub Actions runs unit tests on every push and pull request. Dependencies are resolved from the public `github.com/trains-io/z21.go` module (no local replace).

Tagged releases are built with [GoReleaser](https://goreleaser.com/) when a `v*` tag is pushed.

For command usage and examples, see [README.md](./README.md).
