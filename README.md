# dbmigrate

A PostgreSQL database migration tool with dump, restore, and validation capabilities.

## Installation

```bash
curl -L https://github.com/yourcompany/dbmigrate/releases/latest/download/dbmigrate-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o dbmigrate
chmod +x dbmigrate
sudo mv dbmigrate /usr/local/bin/
```

## Quick Start

1. Create a config file `dbmigrate.yaml`:

```yaml
source:
  host: "staging.db.example.com"
  port: 5432
  database: "mydb"
  user: "dbuser"
  password: "${SRC_PASSWORD}"

target:
  host: "prod.db.example.com"
  port: 5432
  database: "mydb"
  admin_user: "postgres"
  admin_password: "${DST_ADMIN_PASSWORD}"
  app_user: "app_usr"

options:
  parallel_jobs: 4
  data_parallel_jobs: 2
  exclude_tables:
    - "public.activity_log"
```

2. Set environment variables:

```bash
export SRC_PASSWORD="your-source-password"
export DST_ADMIN_PASSWORD="your-target-password"
```

3. Run migration:

```bash
dbmigrate migrate --config dbmigrate.yaml
```

## Commands

- `migrate` - Full migration pipeline
- `dump` - Dump source database
- `restore` - Restore from dump files
- `backup` - Backup target database
- `validate` - Validate migration
- `version` - Show version info

## Documentation

See [docs/](docs/) for detailed documentation.

## License

MIT
