# cloudm-cli

A CLI tool for migrating PostgreSQL databases with dump, restore, and validation capabilities.

## Installation

```bash
curl -L https://github.com/1CL0UD/cloudm-cli/releases/latest/download/cloudm-cli-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o cloudm-cli
chmod +x cloudm-cli
sudo mv cloudm-cli /usr/local/bin/
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

- `cloudm-cli migrate` - Full migration pipeline
- `cloudm-cli dump` - Dump source database
- `cloudm-cli restore` - Restore from dump files
- `cloudm-cli backup` - Backup target database
- `cloudm-cli validate` - Compare source and target
- `cloudm-cli version` - Show version info

## License

MIT
