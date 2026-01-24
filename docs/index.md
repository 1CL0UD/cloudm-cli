# cloudm-cli

A CLI tool for migrating PostgreSQL databases with dump, restore, and validation capabilities.

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/1CL0UD/cloudm-cli/main/install.sh | bash
```

### Manual Download

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -sSL https://github.com/1CL0UD/cloudm-cli/releases/latest/download/cloudm-cli-linux-amd64 -o cloudm-cli

# Linux (arm64)
curl -sSL https://github.com/1CL0UD/cloudm-cli/releases/latest/download/cloudm-cli-linux-arm64 -o cloudm-cli

# macOS (Intel)
curl -sSL https://github.com/1CL0UD/cloudm-cli/releases/latest/download/cloudm-cli-darwin-amd64 -o cloudm-cli

# macOS (Apple Silicon)
curl -sSL https://github.com/1CL0UD/cloudm-cli/releases/latest/download/cloudm-cli-darwin-arm64 -o cloudm-cli

# Install
chmod +x cloudm-cli
sudo mv cloudm-cli /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/1CL0UD/cloudm-cli.git
cd cloudm-cli
make build
sudo cp bin/cloudm-cli /usr/local/bin/
```

## Quick Start

1. Create a config file `db.yaml`:

```yaml
source:
  host: "staging.db.example.com"
  port: 5432
  database: "mydb_staging"
  user: "app_user"
  password: "${SRC_PASSWORD}"

target:
  host: "prod.db.example.com"
  port: 5432
  database: "mydb"
  admin_user: "postgres"
  admin_password: "${DST_ADMIN_PASSWORD}"
  app_user: "app_user"

options:
  parallel_jobs: 4
  data_parallel_jobs: 2
  exclude_tables:
    - "public.activity_log"
  output_dir: "./migrations"
  keep_dumps: true
  extensions:
    - "pg_trgm"
```

2. Run migration:

```bash
export SRC_PASSWORD="your_source_password"
export DST_ADMIN_PASSWORD="your_admin_password"
cloudm-cli migrate --config db.yaml
```

## Commands

| Command               | Description                                                              |
| --------------------- | ------------------------------------------------------------------------ |
| `cloudm-cli migrate`  | Full migration pipeline (backup → dump → restore → ownership → validate) |
| `cloudm-cli dump`     | Dump source database to local files                                      |
| `cloudm-cli restore`  | Restore from existing dump files                                         |
| `cloudm-cli backup`   | Create backup of target database                                         |
| `cloudm-cli validate` | Compare source and target databases                                      |
| `cloudm-cli version`  | Show version information                                                 |

## Global Flags

| Flag         | Description                              |
| ------------ | ---------------------------------------- |
| `--config`   | Path to config file (default: `db.yaml`) |
| `--dry-run`  | Preview operations without executing     |
| `--verbose`  | Enable verbose logging                   |
| `--no-color` | Disable colored output                   |
| `--log-file` | Custom log file path                     |

## Examples

```bash
# Full migration with dry run
cloudm-cli migrate --config db.yaml --dry-run

# Dump only (structure + data)
cloudm-cli dump --config db.yaml --output ./dumps

# Dump structure only
cloudm-cli dump --config db.yaml --structure-only

# Restore from existing dumps
cloudm-cli restore --config db.yaml --input ./migrations/20260119_120000/

# Validate migration
cloudm-cli validate --config db.yaml --detailed

# Create backup of target
cloudm-cli backup --config db.yaml --output ./backups
```

## Requirements

- PostgreSQL client tools (`pg_dump`, `pg_restore`, `psql`)

## License

MIT
