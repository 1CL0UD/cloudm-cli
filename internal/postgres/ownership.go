package postgres

import (
	"context"
	"fmt"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/jackc/pgx/v5"
)

// CreateAppUserIfNotExists creates the app user if it doesn't exist
func CreateAppUserIfNotExists(ctx context.Context, cfg config.TargetConfig, appUser string) error {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Check if user exists
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", appUser).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !exists {
		// Create user with a temporary password
		password := "CHANGE_ME_AFTER_MIGRATION"
		if cfg.AppUserPassword != "" {
			password = cfg.AppUserPassword
		}
		_, err = conn.Exec(ctx, fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", appUser, password))
		if err != nil {
			return fmt.Errorf("failed to create user %s: %w", appUser, err)
		}
	}

	return nil
}

// TransferOwnership transfers ownership of all database objects to app user
func TransferOwnership(ctx context.Context, cfg config.TargetConfig, appUser string) error {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// 1. Alter database owner
	if err := AlterDatabaseOwner(ctx, conn, cfg.Database, appUser); err != nil {
		return err
	}

	// 2. Alter schema owner
	if err := AlterSchemaOwner(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 3. Alter all tables
	if err := AlterTableOwners(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 4. Alter all sequences
	if err := AlterSequenceOwners(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 5. Alter all views
	if err := AlterViewOwners(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 6. Alter all materialized views
	if err := AlterMaterializedViewOwners(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 7. Alter all functions
	if err := AlterFunctionOwners(ctx, conn, "public", appUser); err != nil {
		return err
	}

	// 8. Grant privileges
	if err := GrantPrivileges(ctx, conn, cfg.Database, "public", appUser); err != nil {
		return err
	}

	// 9. Set default privileges
	if err := SetDefaultPrivileges(ctx, conn, "public", appUser); err != nil {
		return err
	}

	return nil
}

// AlterDatabaseOwner changes database owner
func AlterDatabaseOwner(ctx context.Context, conn *pgx.Conn, db, owner string) error {
	_, err := conn.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", db, owner))
	if err != nil {
		return fmt.Errorf("failed to alter database owner: %w", err)
	}
	return nil
}

// AlterSchemaOwner changes schema owner
func AlterSchemaOwner(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	_, err := conn.Exec(ctx, fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s", schema, owner))
	if err != nil {
		return fmt.Errorf("failed to alter schema owner: %w", err)
	}
	return nil
}

// AlterTableOwners changes all table owners in a schema
func AlterTableOwners(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	rows, err := conn.Query(ctx,
		"SELECT tablename FROM pg_tables WHERE schemaname = $1", schema)
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER TABLE %s.%s OWNER TO %s", schema, tableName, owner))
		if err != nil {
			return fmt.Errorf("failed to alter table %s owner: %w", tableName, err)
		}
	}

	return rows.Err()
}

// AlterSequenceOwners changes all sequence owners in a schema
func AlterSequenceOwners(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	rows, err := conn.Query(ctx,
		"SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = $1", schema)
	if err != nil {
		return fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seqName string
		if err := rows.Scan(&seqName); err != nil {
			return fmt.Errorf("failed to scan sequence name: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER SEQUENCE %s.%s OWNER TO %s", schema, seqName, owner))
		if err != nil {
			return fmt.Errorf("failed to alter sequence %s owner: %w", seqName, err)
		}
	}

	return rows.Err()
}

// AlterViewOwners changes all view owners in a schema
func AlterViewOwners(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	rows, err := conn.Query(ctx,
		"SELECT table_name FROM information_schema.views WHERE table_schema = $1", schema)
	if err != nil {
		return fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var viewName string
		if err := rows.Scan(&viewName); err != nil {
			return fmt.Errorf("failed to scan view name: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER VIEW %s.%s OWNER TO %s", schema, viewName, owner))
		if err != nil {
			return fmt.Errorf("failed to alter view %s owner: %w", viewName, err)
		}
	}

	return rows.Err()
}

// AlterMaterializedViewOwners changes all materialized view owners in a schema
func AlterMaterializedViewOwners(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	rows, err := conn.Query(ctx,
		"SELECT matviewname FROM pg_matviews WHERE schemaname = $1", schema)
	if err != nil {
		return fmt.Errorf("failed to query materialized views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var matviewName string
		if err := rows.Scan(&matviewName); err != nil {
			return fmt.Errorf("failed to scan materialized view name: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER MATERIALIZED VIEW %s.%s OWNER TO %s", schema, matviewName, owner))
		if err != nil {
			return fmt.Errorf("failed to alter materialized view %s owner: %w", matviewName, err)
		}
	}

	return rows.Err()
}

// AlterFunctionOwners changes all function owners in a schema
func AlterFunctionOwners(ctx context.Context, conn *pgx.Conn, schema, owner string) error {
	rows, err := conn.Query(ctx, `
		SELECT p.proname, pg_get_function_identity_arguments(p.oid) 
		FROM pg_proc p 
		JOIN pg_namespace n ON n.oid = p.pronamespace 
		WHERE n.nspname = $1`, schema)
	if err != nil {
		return fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var funcName, args string
		if err := rows.Scan(&funcName, &args); err != nil {
			return fmt.Errorf("failed to scan function info: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER FUNCTION %s.%s(%s) OWNER TO %s", schema, funcName, args, owner))
		if err != nil {
			return fmt.Errorf("failed to alter function %s owner: %w", funcName, err)
		}
	}

	return rows.Err()
}

// GrantPrivileges grants all privileges to app user
func GrantPrivileges(ctx context.Context, conn *pgx.Conn, db, schema, user string) error {
	grants := []string{
		fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", db, user),
		fmt.Sprintf("GRANT ALL ON SCHEMA %s TO %s", schema, user),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA %s TO %s", schema, user),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA %s TO %s", schema, user),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA %s TO %s", schema, user),
	}

	for _, grant := range grants {
		_, err := conn.Exec(ctx, grant)
		if err != nil {
			return fmt.Errorf("failed to execute grant: %w", err)
		}
	}

	return nil
}

// SetDefaultPrivileges sets default privileges for future objects
func SetDefaultPrivileges(ctx context.Context, conn *pgx.Conn, schema, user string) error {
	defaults := []string{
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s", schema, user),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s", schema, user),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON FUNCTIONS TO %s", schema, user),
	}

	for _, def := range defaults {
		_, err := conn.Exec(ctx, def)
		if err != nil {
			return fmt.Errorf("failed to set default privileges: %w", err)
		}
	}

	return nil
}