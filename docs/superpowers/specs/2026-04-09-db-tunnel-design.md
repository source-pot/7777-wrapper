# db-tunnel Design Spec

## Overview

A single-binary Go CLI that reads a TOML config file, presents an interactive database picker using promptui, and invokes `7777` with the selected database's parameters.

## Project Structure

```
/Users/robwatson/Developer/7777/
├── main.go              # Entry point: load config, picker, exec 7777
├── config.go            # TOML config parsing and validation
├── go.mod
├── go.sum
├── config.example.toml  # Example config with dummy values
└── README.md
```

## Config Format (TOML)

Global defaults at the top level. Per-database entries override globals.

```toml
aws_profile = "default"
aws_region = "ap-southeast-2"

[[database]]
name = "Production Main"
database = "classmanager-prod-master"
port = 7700

[[database]]
name = "ClassManager V3 (EU)"
database = "classmanager-v3"
port = 7720
aws_profile = "classmanager-v3"
aws_region = "eu-west-2"
```

Config file location: `config.toml` in the same directory as the binary, resolved via `os.Executable()`.

### Fields

**Global (optional):**
- `aws_profile` — default AWS profile for all databases
- `aws_region` — default AWS region for all databases

**Per-database (required unless noted):**
- `name` — human-friendly label shown in the picker (required)
- `database` — RDS database identifier passed to 7777 (required)
- `port` — local port for the tunnel (required)
- `aws_profile` — overrides global default (optional)
- `aws_region` — overrides global default (optional)

## Application Flow

1. **Load config** — find and parse `config.toml` next to binary
2. **Validate** — ensure at least one database entry; each must have `name`, `database`, and `port`
3. **Present picker** — promptui Select showing database `name` values
4. **Merge defaults** — fill in `aws_profile`/`aws_region` from global defaults where not set per-database
5. **Exec 7777** — run: `7777 --port=<port> --database=<db> --profile=<profile> --region=<region>`
6. **Wait** — process stays alive, forwarding stdout/stderr, until 7777 exits or user hits Ctrl+C

## 7777 Invocation

Using `os/exec` with `cmd.Run()`:
- `--port`, `--database`, `--profile`, `--region` passed as CLI flags
- stdout and stderr piped to the terminal
- Ctrl+C (SIGINT) caught and forwarded to the 7777 child process so it cleans up properly
- db-tunnel exits when 7777 exits

## Error Handling

| Scenario | Behaviour |
|---|---|
| `config.toml` not found | Print: "config.toml not found next to binary, see config.example.toml" |
| TOML parse error | Print the parse error |
| No databases configured | Print: "no databases configured in config.toml" |
| `7777` not found in PATH | Print: "7777 command not found in PATH" |
| User cancels picker (Ctrl+C) | Clean exit, no error message |

## Dependencies

- `github.com/BurntSushi/toml` — TOML parsing
- `github.com/manifoldco/promptui` — interactive list picker
- Go standard library: `os`, `os/exec`, `os/signal`, `path/filepath`, `fmt`

## Out of Scope

- Config file creation/wizard
- Multi-database selection (run multiple tunnels simultaneously)
- Extra 7777 flags (ttl, forever, verbose, security-group, subnet)
