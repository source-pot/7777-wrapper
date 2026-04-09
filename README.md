# db-tunnel

An interactive CLI wrapper for [7777](https://port7777.com/) that lets you select from pre-configured database tunnels.

## Prerequisites

- [7777](https://port7777.com/) installed and configured
- AWS credentials configured for your profile(s)

## Installation

Build from source:

    go build -o db-tunnel .

Place the `db-tunnel` binary wherever you like (e.g. `/usr/local/bin/`).

## Configuration

Create a `config.toml` file in the same directory as the `db-tunnel` binary. See `config.example.toml` for all available options.

    cp config.example.toml config.toml

### Config format

```toml
# Global defaults (applied when not specified per-database)
aws_profile = "default"
aws_region = "ap-southeast-2"

# Each [[database]] block defines a tunnel you can connect to.
[[database]]
name = "Production Main"          # Shown in the picker
database = "myapp-prod-master"    # RDS identifier passed to 7777
port = 7700                       # Local port for the tunnel

[[database]]
name = "Other Project"
database = "other-db"
port = 7720
aws_profile = "other-profile"    # Overrides global default
aws_region = "us-east-1"         # Overrides global default
```

### Fields

**Global (optional):**

| Field | Description |
|---|---|
| `aws_profile` | Default AWS profile for all databases |
| `aws_region` | Default AWS region for all databases |

**Per-database:**

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Label shown in the interactive picker |
| `database` | Yes | RDS database identifier passed to 7777 |
| `port` | Yes | Local port for the tunnel |
| `aws_profile` | No | Overrides global `aws_profile` |
| `aws_region` | No | Overrides global `aws_region` |

## Usage

Run the binary:

    ./db-tunnel

Use the arrow keys to select a database, then press Enter. The 7777 tunnel will start and remain open until you press `Ctrl+C`.

## How it works

db-tunnel reads your config, presents an interactive picker, then runs:

    7777 --port=<port> --database=<database> --profile=<profile> --region=<region>

The tunnel stays open in the foreground. Press `Ctrl+C` to close it — 7777 will clean up its Fargate bastion container automatically.

To connect to multiple databases simultaneously, run db-tunnel in separate terminal windows.
