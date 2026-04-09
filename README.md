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

db-tunnel looks for a config file in the following locations (first found wins):

1. `~/.config/db-tunnel/config.toml`
2. `~/.db-tunnel.toml`
3. `config.toml` next to the `db-tunnel` binary

The recommended location is `~/.config/db-tunnel/config.toml`:

    mkdir -p ~/.config/db-tunnel
    cp config.example.toml ~/.config/db-tunnel/config.toml

### Config format

```toml
# Global defaults (applied when not specified per-database)
aws_profile = "default"
aws_region = "ap-southeast-2"

# Global-only settings
verbose = true
license = "your-license-key"

# Global defaults with per-database override support
security_group = "sg-0123456789abcdef0"
subnet = "subnet-0123456789abcdef0"
ttl = 2
forever = false

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
security_group = "sg-other"      # Overrides global default
forever = true                   # Overrides global default
```

### Fields

**Global only:**

| Field | Description |
|---|---|
| `verbose` | Enable verbose 7777 output |
| `license` | 7777 license code |

**Global default with per-database override:**

| Field | Description |
|---|---|
| `aws_profile` | AWS profile |
| `aws_region` | AWS region |
| `security_group` | Fargate security group |
| `subnet` | Fargate subnet |
| `ttl` | Tunnel time-to-live in hours (7777 default: 2) |
| `forever` | Ignore TTL and run the tunnel indefinitely |

**Per-database only:**

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Label shown in the interactive picker |
| `database` | Yes | RDS database identifier passed to 7777 |
| `port` | Yes | Local port for the tunnel |
| `elasticache` | No | Connect to ElastiCache instead of RDS |

## Usage

Run the binary:

    ./db-tunnel

Use the arrow keys to select a database, then press Enter. The 7777 tunnel will start and remain open until you press `Ctrl+C`.

## How it works

db-tunnel reads your config, presents an interactive picker, then runs:

    7777 --port=<port> --database=<database> --profile=<profile> --region=<region>

The tunnel stays open in the foreground. Press `Ctrl+C` to close it — 7777 will clean up its Fargate bastion container automatically.

To connect to multiple databases simultaneously, run db-tunnel in separate terminal windows.
