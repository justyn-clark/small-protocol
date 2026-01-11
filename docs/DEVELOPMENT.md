# Development Guide

This guide covers building and testing the SMALL CLI during development.

## Building the CLI

### Quick Build

Build a local binary without installing globally:

```bash
go build -o ./bin/small ./cmd/small
./bin/small --help
```

### Building After Code Changes

After modifying CLI code, you must rebuild before changes take effect:

```bash
go build -o ./bin/small ./cmd/small
./bin/small [command]
```

**Important:** The binary is statically linked. Changes to Go source files require a rebuild.

## Testing

Run the full test suite:

```bash
go test ./...
```

Run tests for a specific package:

```bash
go test ./internal/commands -v
go test ./internal/small -v
```

Run a specific test:

```bash
go test -run TestGenerateReplayId ./internal/commands -v
```

## Installing Globally During Development

To use `small` commands from anywhere during development, install to your `$GOPATH/bin`:

```bash
go build -o ~/go/bin/small ./cmd/small
```

Verify installation:

```bash
which small
small --help
```

### Updating After Code Changes

After making changes and rebuilding, update the global binary:

```bash
go build -o ~/go/bin/small ./cmd/small
```

**Note on shell caching:** If you've already run `small` in your shell session, it may cache the old binary location. Clear the cache:

```bash
hash -r
small --help
```

### PATH Order Issue

If `small` shows old behavior after updating:

1. Check which binary is being used:
   ```bash
   which small
   ```

2. If multiple versions exist, your `PATH` order matters. Your `~/go/bin` typically comes before `/usr/local/bin`.

3. Always build to your active `$GOPATH/bin` (usually `~/go/bin/small`), not `/usr/local/bin`.

## Schema Updates

If you modify schemas in `spec/small/v1.0.0/schemas/`, you must sync them to the embedded location:

```bash
make sync-schemas
```

Then rebuild:

```bash
go build -o ~/go/bin/small ./cmd/small
```

## Validation During Development

Before committing or testing:

```bash
# Validate all artifacts
small validate

# Check invariants
small lint --strict

# Verify CI gates pass
small verify --ci
```

## Common Tasks

### Add a New Command

1. Create `internal/commands/mycommand.go` with a `mycommandCmd()` function
2. Register it in `internal/commands/root.go`:
   ```go
   rootCmd.AddCommand(mycommandCmd())
   ```
3. Add tests in `internal/commands/mycommand_test.go`
4. Rebuild: `go build -o ~/go/bin/small ./cmd/small`

### Modify Schemas

1. Edit `spec/small/v1.0.0/schemas/*.json`
2. Sync to embedded copy: `make sync-schemas`
3. Update tests if behavior changed
4. Rebuild: `go build -o ~/go/bin/small ./cmd/small`
5. Run `go test ./...` to verify

### Test a Local Binary

Use the local binary without installing globally:

```bash
./bin/small init --intent "test"
./bin/small validate
./bin/small handoff
```

## Makefile Commands

Common development targets:

```bash
make small-build       # Build the CLI
make small-test        # Run all tests
make small-format      # Format code
make small-validate    # Validate examples
make sync-schemas      # Sync schemas to embedded location
```

See the Makefile for complete list.
