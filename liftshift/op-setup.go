package liftshift

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CreateAppUserIfNotExists creates the application user if it doesn't exist
func CreateAppUserIfNotExists(config *Config) error {
	fmt.Println("Checking if user exists and creating if needed...")

	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DstHost, config.DstPort, config.DstAdminUser, config.DstAdminPassword, config.DstDB)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer conn.Close(ctx)

	// Check if user exists
	var userExists int
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s' LIMIT 1;", config.AppUser)).Scan(&userExists)
	if err != nil && err.Error() != "no rows in result set" {
		// If there's an error other than "no rows", return it
		if err.Error() != "no rows in result set" {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}
	}

	if err.Error() == "no rows in result set" {
		// User doesn't exist, create it
		fmt.Printf("Creating user %s...\n", config.AppUser)
		createUserQuery := fmt.Sprintf("CREATE USER %s WITH PASSWORD 'CHANGE_ME_AFTER_MIGRATION';", config.AppUser)
		_, err = conn.Exec(ctx, createUserQuery)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		fmt.Printf("User %s created. IMPORTANT: Change password after migration!\n", config.AppUser)
	} else {
		fmt.Printf("User %s already exists.\n", config.AppUser)
	}

	return nil
}

// TransferOwnership transfers ownership of database objects to the application user
func TransferOwnership(config *Config) error {
	fmt.Printf("Transferring ownership to %s...\n", config.AppUser)

	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DstHost, config.DstPort, config.DstAdminUser, config.DstAdminPassword, config.DstDB)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer conn.Close(ctx)

	queries := []string{
		// 1. Alter database owner
		fmt.Sprintf("ALTER DATABASE %s OWNER TO %s;", config.DstDB, config.AppUser),

		// 2. Alter schema owner
		fmt.Sprintf("ALTER SCHEMA public OWNER TO %s;", config.AppUser),

		// 3. Alter all tables
		fmt.Sprintf(`
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER TABLE public.%%I OWNER TO %s;', r.tablename);
    END LOOP;
END
$$;`, config.AppUser),

		// 4. Alter all sequences
		fmt.Sprintf(`
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT sequence_name
        FROM information_schema.sequences
        WHERE sequence_schema = 'public'
    LOOP
        EXECUTE format('ALTER SEQUENCE public.%%I OWNER TO %s;', r.sequence_name);
    END LOOP;
END
$$;`, config.AppUser),

		// 5. Alter all views
		fmt.Sprintf(`
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT table_name
        FROM information_schema.views
        WHERE table_schema = 'public'
    LOOP
        EXECUTE format('ALTER VIEW public.%%I OWNER TO %s;', r.table_name);
    END LOOP;
END
$$;`, config.AppUser),

		// 6. Alter all materialized views
		fmt.Sprintf(`
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT matviewname
        FROM pg_matviews
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER MATERIALIZED VIEW public.%%I OWNER TO %s;', r.matviewname);
    END LOOP;
END
$$;`, config.AppUser),

		// 7. Alter all functions
		fmt.Sprintf(`
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT p.oid,
               n.nspname AS schema_name,
               p.proname AS func_name,
               pg_get_function_identity_arguments(p.oid) AS args
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'public'
    LOOP
        EXECUTE format(
            'ALTER FUNCTION %%I.%%I(%%s) OWNER TO %s;',
            r.schema_name,
            r.func_name,
            r.args
        );
    END LOOP;
END
$$;`, config.AppUser),

		// 8. Grant all privileges
		fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", config.DstDB, config.AppUser),
		fmt.Sprintf("GRANT ALL ON SCHEMA public TO %s;", config.AppUser),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES    IN SCHEMA public TO %s;", config.AppUser),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO %s;", config.AppUser),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO %s;", config.AppUser),

		// 9. Set default privileges for future objects
		fmt.Sprintf(`
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON TABLES TO %s;`, config.AppUser),

		fmt.Sprintf(`
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON SEQUENCES TO %s;`, config.AppUser),

		fmt.Sprintf(`
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON FUNCTIONS TO %s;`, config.AppUser),
	}

	for _, query := range queries {
		_, err := conn.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to execute ownership query: %w", err)
		}
	}

	fmt.Printf("Ownership configured successfully for %s\n", config.AppUser)
	return nil
}

// ConfigureOwnership sets up the application user and transfers ownership
func ConfigureOwnership(config *Config) error {
	err := CreateAppUserIfNotExists(config)
	if err != nil {
		return fmt.Errorf("failed to create application user: %w", err)
	}

	err = TransferOwnership(config)
	if err != nil {
		return fmt.Errorf("failed to transfer ownership: %w", err)
	}

	return nil
}
