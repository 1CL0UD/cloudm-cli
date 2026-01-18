#!/usr/bin/env bash
set -euo pipefail

# ============================================
# PostgreSQL Database Migration Script
# Source: Staging -> Target: RDS Production
# ============================================

# ============ CONFIGURATION ============
# Source (Staging)
SRC_HOST="${SRC_HOST:-redacted}"
SRC_PORT="${SRC_PORT:-5432}"
SRC_DB="${SRC_DB:-apoadbmasterstaging}"
SRC_USER="${SRC_USER:-appapoaprod}"
SRC_PASSWORD="${SRC_PASSWORD:-}"  # Set via environment variable or .pgpass

# Target (RDS Production)
DST_HOST="${DST_HOST:-redacted}"
DST_PORT="${DST_PORT:-5432}"
DST_DB="${DST_DB:-apoa}"
DST_ADMIN_USER="${DST_ADMIN_USER:-postgres}"
DST_ADMIN_PASSWORD="${DST_ADMIN_PASSWORD:-}"  # Set via environment variable or .pgpass
APP_USER="${APP_USER:-app_usr}"

# File names
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
STRUCT_DUMP="structure_${TIMESTAMP}.dump"
DATA_DUMP="data_${TIMESTAMP}.dump"
BACKUP_DUMP="backup_pre_migration_${TIMESTAMP}.dump"
TIME_LOG="migration_time_${TIMESTAMP}.txt"
MAIN_LOG="migration_${TIMESTAMP}.log"
VALIDATION_LOG="validation_${TIMESTAMP}.log"

# Options
DRY_RUN="${DRY_RUN:-false}"
SKIP_BACKUP="${SKIP_BACKUP:-false}"
PARALLEL_JOBS="${PARALLEL_JOBS:-4}"
DATA_PARALLEL_JOBS="${DATA_PARALLEL_JOBS:-2}"

# ============ FUNCTIONS ============

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "${MAIN_LOG}"
}

log_error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "${MAIN_LOG}" >&2
}

check_requirements() {
    log "Checking requirements..."
    
    local missing_tools=()
    for tool in pg_dump pg_restore psql; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    if [ -z "${SRC_PASSWORD}" ]; then
        log_error "SRC_PASSWORD not set. Set via environment variable or configure .pgpass"
        exit 1
    fi
    
    if [ -z "${DST_ADMIN_PASSWORD}" ]; then
        log_error "DST_ADMIN_PASSWORD not set. Set via environment variable or configure .pgpass"
        exit 1
    fi
    
    log "Requirements check passed."
}

test_connection() {
    local host=$1
    local port=$2
    local user=$3
    local db=$4
    local password=$5
    local name=$6
    
    log "Testing connection to ${name}..."
    export PGPASSWORD="${password}"
    
    if ! psql -h "${host}" -p "${port}" -U "${user}" -d "${db}" -c "SELECT 1;" &>/dev/null; then
        log_error "Cannot connect to ${name} (${host}:${port}/${db})"
        return 1
    fi
    
    log "Connection to ${name} successful."
    return 0
}

calculate_elapsed_time() {
    local start=$1
    local end=$2
    local elapsed=$((end - start))
    local hours=$((elapsed / 3600))
    local minutes=$(((elapsed % 3600) / 60))
    local seconds=$((elapsed % 60))
    
    printf "%02d:%02d:%02d" $hours $minutes $seconds
}

terminate_connections() {
    log "Terminating active connections to ${DST_DB}..."
    export PGPASSWORD="${DST_ADMIN_PASSWORD}"
    
    psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "postgres" \
        -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DST_DB}' AND pid <> pg_backend_pid();" \
        2>>"${MAIN_LOG}" || {
            log_error "Failed to terminate connections"
            return 1
        }
    
    log "Connections terminated."
}

create_app_user_if_not_exists() {
    log "Checking if user ${APP_USER} exists..."
    export PGPASSWORD="${DST_ADMIN_PASSWORD}"
    
    local user_exists=$(psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
        -tAc "SELECT 1 FROM pg_roles WHERE rolname='${APP_USER}';" 2>>"${MAIN_LOG}")
    
    if [ -z "$user_exists" ]; then
        log "Creating user ${APP_USER}..."
        psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
            -c "CREATE USER ${APP_USER} WITH PASSWORD 'CHANGE_ME_AFTER_MIGRATION';" \
            2>>"${MAIN_LOG}" || {
                log_error "Failed to create user ${APP_USER}"
                return 1
            }
        log "User ${APP_USER} created. IMPORTANT: Change password after migration!"
    else
        log "User ${APP_USER} already exists."
    fi
}

validate_migration() {
    log "Validating migration..."
    export PGPASSWORD="${SRC_PASSWORD}"
    
    # Get source table counts
    psql -h "${SRC_HOST}" -p "${SRC_PORT}" -U "${SRC_USER}" -d "${SRC_DB}" \
        -tAc "SELECT schemaname, tablename, n_live_tup FROM pg_stat_user_tables WHERE schemaname='public' ORDER BY tablename;" \
        > "/tmp/src_counts_${TIMESTAMP}.txt" 2>>"${MAIN_LOG}"
    
    export PGPASSWORD="${DST_ADMIN_PASSWORD}"
    
    # Get target table counts
    psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
        -tAc "SELECT schemaname, tablename, n_live_tup FROM pg_stat_user_tables WHERE schemaname='public' ORDER BY tablename;" \
        > "/tmp/dst_counts_${TIMESTAMP}.txt" 2>>"${MAIN_LOG}"
    
    # Compare and log
    {
        echo "=========================================="
        echo "Migration Validation Report"
        echo "Generated: $(date)"
        echo "=========================================="
        echo ""
        echo "Source Database: ${SRC_DB}"
        cat "/tmp/src_counts_${TIMESTAMP}.txt"
        echo ""
        echo "Target Database: ${DST_DB}"
        cat "/tmp/dst_counts_${TIMESTAMP}.txt"
        echo ""
        echo "Database Size:"
        psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
            -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size FROM pg_tables WHERE schemaname='public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;" \
            2>>"${MAIN_LOG}"
    } > "${VALIDATION_LOG}"
    
    log "Validation report saved to ${VALIDATION_LOG}"
    rm -f "/tmp/src_counts_${TIMESTAMP}.txt" "/tmp/dst_counts_${TIMESTAMP}.txt"
}

# ============ MAIN SCRIPT ============

log "==== POSTGRESQL DATABASE MIGRATION ===="
log "Source: ${SRC_HOST}/${SRC_DB}"
log "Target: ${DST_HOST}/${DST_DB}"
log "Dry Run: ${DRY_RUN}"
log ""

START_TIME=$(date +%s)

# Pre-flight checks
check_requirements
test_connection "${SRC_HOST}" "${SRC_PORT}" "${SRC_USER}" "${SRC_DB}" "${SRC_PASSWORD}" "SOURCE" || exit 1
test_connection "${DST_HOST}" "${DST_PORT}" "${DST_ADMIN_USER}" "${DST_DB}" "${DST_ADMIN_PASSWORD}" "TARGET" || exit 1

if [ "${DRY_RUN}" = "true" ]; then
    log "DRY RUN MODE - No changes will be made"
    log "Script validation completed successfully"
    exit 0
fi

########################################
# STEP 0: BACKUP TARGET DATABASE
########################################
if [ "${SKIP_BACKUP}" = "false" ]; then
    log "== STEP 0: Backup target database =="
    export PGPASSWORD="${DST_ADMIN_PASSWORD}"
    
    BACKUP_START=$(date +%s)
    pg_dump \
        -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
        -Fc \
        -f "${BACKUP_DUMP}" \
        2>>"${MAIN_LOG}" || {
            log_error "Backup failed"
            exit 1
        }
    BACKUP_END=$(date +%s)
    
    log "Backup completed in $(calculate_elapsed_time $BACKUP_START $BACKUP_END)"
    log "Backup saved to: ${BACKUP_DUMP}"
else
    log "== STEP 0: Skipping backup (SKIP_BACKUP=true) =="
fi

########################################
# STEP 1: DUMP FROM SOURCE
########################################
log "== STEP 1: Dump from ${SRC_DB} =="
export PGPASSWORD="${SRC_PASSWORD}"

DUMP_START=$(date +%s)

# Dump structure
log "Dumping database structure..."
pg_dump \
    -h "${SRC_HOST}" -p "${SRC_PORT}" -U "${SRC_USER}" -d "${SRC_DB}" \
    -n public -s \
    -Fc \
    -f "${STRUCT_DUMP}" \
    2>>"${MAIN_LOG}" || {
        log_error "Structure dump failed"
        exit 1
    }

log "Structure dump completed: ${STRUCT_DUMP}"

# Dump data
log "Dumping database data (excluding activity_log)..."
pg_dump \
    -h "${SRC_HOST}" -p "${SRC_PORT}" -U "${SRC_USER}" -d "${SRC_DB}" \
    -n public -a \
    --exclude-table=public.activity_log \
    -Fc \
    -f "${DATA_DUMP}" \
    2>>"${MAIN_LOG}" || {
        log_error "Data dump failed"
        exit 1
    }

DUMP_END=$(date +%s)
log "Data dump completed: ${DATA_DUMP}"
log "Dump phase completed in $(calculate_elapsed_time $DUMP_START $DUMP_END)"

########################################
# STEP 2: CLEAN & RESTORE TO TARGET
########################################
log "== STEP 2: Clean & restore to ${DST_DB} =="

export PGPASSWORD="${DST_ADMIN_PASSWORD}"

RESTORE_START=$(date +%s)

# Terminate active connections
terminate_connections || exit 1

# Drop & recreate schema
log "Dropping schema public..."
psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
    -c "DROP SCHEMA IF EXISTS public CASCADE;" \
    2>>"${MAIN_LOG}" || {
        log_error "DROP SCHEMA failed"
        exit 1
    }

log "Creating schema public..."
psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
    -c "CREATE SCHEMA public;" \
    2>>"${MAIN_LOG}" || {
        log_error "CREATE SCHEMA failed"
        exit 1
    }

# Create extensions before structure restore
log "Creating pg_trgm extension..."
psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" \
    -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;" \
    2>>"${MAIN_LOG}" || {
        log_error "CREATE EXTENSION pg_trgm failed"
        exit 1
    }

# Restore structure
log "Restoring database structure (parallel jobs: ${PARALLEL_JOBS})..."
pg_restore -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" \
    -d "${DST_DB}" -n public -s -j "${PARALLEL_JOBS}" \
    --no-owner --no-privileges \
    "${STRUCT_DUMP}" \
    2>>"${MAIN_LOG}" || {
        log_error "Structure restore failed"
        exit 1
    }

log "Structure restored successfully"

# Restore data
log "Restoring database data (parallel jobs: ${DATA_PARALLEL_JOBS})..."
pg_restore -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" \
    -d "${DST_DB}" -n public -a -j "${DATA_PARALLEL_JOBS}" \
    --no-owner --no-privileges \
    "${DATA_DUMP}" \
    2>>"${MAIN_LOG}" || {
        log_error "Data restore failed"
        exit 1
    }

RESTORE_END=$(date +%s)
log "Data restored successfully"
log "Restore phase completed in $(calculate_elapsed_time $RESTORE_START $RESTORE_END)"

########################################
# STEP 3: CREATE APP USER & SET OWNERSHIP
########################################
log "== STEP 3: Configure ownership for ${APP_USER} =="

OWNERSHIP_START=$(date +%s)

create_app_user_if_not_exists || exit 1

log "Transferring ownership to ${APP_USER}..."

psql -h "${DST_HOST}" -p "${DST_PORT}" -U "${DST_ADMIN_USER}" -d "${DST_DB}" <<SQL >>"${MAIN_LOG}" 2>&1

-- 1. Alter database owner
ALTER DATABASE ${DST_DB} OWNER TO ${APP_USER};

-- 2. Alter schema owner
ALTER SCHEMA public OWNER TO ${APP_USER};

-- 3. Alter all tables
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO ${APP_USER};', r.tablename);
    END LOOP;
END
\$\$;

-- 4. Alter all sequences
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT sequence_name
        FROM information_schema.sequences
        WHERE sequence_schema = 'public'
    LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO ${APP_USER};', r.sequence_name);
    END LOOP;
END
\$\$;

-- 5. Alter all views
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT table_name
        FROM information_schema.views
        WHERE table_schema = 'public'
    LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO ${APP_USER};', r.table_name);
    END LOOP;
END
\$\$;

-- 6. Alter all materialized views
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT matviewname
        FROM pg_matviews
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER MATERIALIZED VIEW public.%I OWNER TO ${APP_USER};', r.matviewname);
    END LOOP;
END
\$\$;

-- 7. Alter all functions
DO \$\$
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
            'ALTER FUNCTION %I.%I(%s) OWNER TO ${APP_USER};',
            r.schema_name,
            r.func_name,
            r.args
        );
    END LOOP;
END
\$\$;

-- 8. Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE ${DST_DB} TO ${APP_USER};
GRANT ALL ON SCHEMA public TO ${APP_USER};
GRANT ALL PRIVILEGES ON ALL TABLES    IN SCHEMA public TO ${APP_USER};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ${APP_USER};
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO ${APP_USER};

-- 9. Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON TABLES TO ${APP_USER};

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON SEQUENCES TO ${APP_USER};

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL ON FUNCTIONS TO ${APP_USER};

SQL

if [ $? -ne 0 ]; then
    log_error "Ownership transfer failed"
    exit 1
fi

OWNERSHIP_END=$(date +%s)
log "Ownership configured successfully"
log "Ownership phase completed in $(calculate_elapsed_time $OWNERSHIP_START $OWNERSHIP_END)"

########################################
# STEP 4: VALIDATION
########################################
log "== STEP 4: Validation =="

validate_migration

########################################
# SUMMARY
########################################
END_TIME=$(date +%s)
TOTAL_ELAPSED=$(calculate_elapsed_time $START_TIME $END_TIME)

{
    echo "=========================================="
    echo "Migration Timing Summary"
    echo "=========================================="
    echo "Start Time: $(date -d @${START_TIME} +'%Y-%m-%d %H:%M:%S')"
    echo "End Time: $(date -d @${END_TIME} +'%Y-%m-%d %H:%M:%S')"
    echo "Total Duration: ${TOTAL_ELAPSED}"
    echo ""
    echo "Phase Breakdown:"
    if [ "${SKIP_BACKUP}" = "false" ]; then
        echo "  Backup: $(calculate_elapsed_time $BACKUP_START $BACKUP_END)"
    fi
    echo "  Dump: $(calculate_elapsed_time $DUMP_START $DUMP_END)"
    echo "  Restore: $(calculate_elapsed_time $RESTORE_START $RESTORE_END)"
    echo "  Ownership: $(calculate_elapsed_time $OWNERSHIP_START $OWNERSHIP_END)"
    echo ""
    echo "Files Generated:"
    echo "  Structure: ${STRUCT_DUMP}"
    echo "  Data: ${DATA_DUMP}"
    if [ "${SKIP_BACKUP}" = "false" ]; then
        echo "  Backup: ${BACKUP_DUMP}"
    fi
    echo "  Main Log: ${MAIN_LOG}"
    echo "  Validation: ${VALIDATION_LOG}"
} | tee -a "${TIME_LOG}"

log ""
log "==== MIGRATION COMPLETED SUCCESSFULLY ===="
log "Total time: ${TOTAL_ELAPSED}"
log "Review logs: ${MAIN_LOG}, ${TIME_LOG}, ${VALIDATION_LOG}"

if [ -f "${BACKUP_DUMP}" ]; then
    log ""
    log "IMPORTANT: Pre-migration backup saved to ${BACKUP_DUMP}"
    log "Keep this backup until you've verified the migration is successful"
fi

log ""
log "Next steps:"
log "1. Review validation report: ${VALIDATION_LOG}"
log "2. Test application connectivity"
log "3. If ${APP_USER} was created, change its password"
log "4. Verify critical business processes"
log "5. Once verified, clean up dump files and old backup"