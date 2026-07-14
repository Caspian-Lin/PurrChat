// Package migrate applies checked, append-only PostgreSQL migrations.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	filePattern    = regexp.MustCompile(`^([0-9]+)_[a-z0-9][a-z0-9_-]*\.sql$`)
	versionPattern = regexp.MustCompile(`^([0-9]+)([a-z][a-z0-9-]*)?$`)
)

// Config declares the migration ownership and legacy filename aliases for one service.
type Config struct {
	Service        string
	Dir            string
	BaselineTable  string
	LegacyVersions map[string]string
}

// Migration is an immutable SQL file and its canonical, service-scoped version.
type Migration struct {
	Version  string
	Filename string
	SQL      string
	Checksum string
	sequence int64
	suffix   string
}

// Runner applies one service's migrations against a shared PostgreSQL database.
type Runner struct {
	config Config
}

// NewRunner creates a migration runner. Validation occurs when migrations are loaded.
func NewRunner(config Config) *Runner {
	return &Runner{config: config}
}

// Discover verifies and orders the SQL files without touching the database.
func (r *Runner) Discover() ([]Migration, error) {
	if r.config.Service == "" {
		return nil, fmt.Errorf("migration service is required")
	}
	if r.config.Dir == "" {
		return nil, fmt.Errorf("migration directory is required")
	}

	entries, err := os.ReadDir(r.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("read migration directory %q: %w", r.config.Dir, err)
	}

	seenFiles := make(map[string]bool, len(entries))
	seenVersions := make(map[string]string, len(entries))
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		filename := entry.Name()
		matches := filePattern.FindStringSubmatch(filename)
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename %q: use NNN_description.sql", filename)
		}
		seenFiles[filename] = true

		version := matches[1]
		if legacyVersion, ok := r.config.LegacyVersions[filename]; ok {
			version = legacyVersion
		}
		sequence, suffix, err := parseVersion(version)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version %q for %s: %w", version, filename, err)
		}
		if previous, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %q in %s and %s", version, previous, filename)
		}

		content, err := os.ReadFile(filepath.Join(r.config.Dir, filename))
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", filename, err)
		}
		digest := sha256.Sum256(content)
		seenVersions[version] = filename
		migrations = append(migrations, Migration{
			Version:  version,
			Filename: filename,
			SQL:      string(content),
			Checksum: hex.EncodeToString(digest[:]),
			sequence: sequence,
			suffix:   suffix,
		})
	}

	for filename := range r.config.LegacyVersions {
		if !seenFiles[filename] {
			return nil, fmt.Errorf("legacy migration mapping references missing file %q", filename)
		}
	}
	if len(migrations) == 0 {
		return nil, fmt.Errorf("no SQL migrations found in %q", r.config.Dir)
	}

	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].sequence != migrations[j].sequence {
			return migrations[i].sequence < migrations[j].sequence
		}
		return migrations[i].suffix < migrations[j].suffix
	})
	return migrations, nil
}

// Run applies pending migrations once. Existing databases without a migration record must
// be explicitly baselined instead of being guessed or replayed.
func (r *Runner) Run(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := r.Discover()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if err := r.lock(ctx, conn); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", r.lockName())
	}()

	if err := ensureMigrationTable(ctx, conn); err != nil {
		return err
	}
	applied, err := r.verifyApplied(ctx, conn, migrations)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		exists, err := r.baselineTableExists(ctx, conn)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%s schema exists without migration history; run %q to record an explicit baseline", r.config.Service, "migrate baseline")
		}
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := applyMigration(ctx, conn, r.config.Service, migration); err != nil {
			return err
		}
	}
	return nil
}

// Baseline records the current schema without executing SQL. It is intentionally explicit
// and only succeeds when the service's sentinel table already exists.
func (r *Runner) Baseline(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := r.Discover()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()
	if err := r.lock(ctx, conn); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", r.lockName())
	}()

	if err := ensureMigrationTable(ctx, conn); err != nil {
		return err
	}
	applied, err := r.verifyApplied(ctx, conn, migrations)
	if err != nil {
		return err
	}
	if len(applied) != 0 {
		return fmt.Errorf("%s already has migration history; refusing to overwrite it", r.config.Service)
	}
	exists, err := r.baselineTableExists(ctx, conn)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("cannot baseline %s: expected existing table %q was not found", r.config.Service, r.config.BaselineTable)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin baseline transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, migration := range migrations {
		if _, err := tx.Exec(ctx,
			`INSERT INTO purrchat_schema_migrations (service, version, filename, checksum)
			 VALUES ($1, $2, $3, $4)`,
			r.config.Service, migration.Version, migration.Filename, migration.Checksum); err != nil {
			return fmt.Errorf("record baseline migration %s: %w", migration.Filename, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit baseline: %w", err)
	}
	return nil
}

func parseVersion(version string) (int64, string, error) {
	match := versionPattern.FindStringSubmatch(version)
	if match == nil {
		return 0, "", fmt.Errorf("use a numeric version with an optional lowercase legacy suffix")
	}
	sequence, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || sequence < 1 {
		return 0, "", fmt.Errorf("numeric version must be positive")
	}
	return sequence, match[2], nil
}

func (r *Runner) lock(ctx context.Context, conn *pgxpool.Conn) error {
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext($1))", r.lockName()); err != nil {
		return fmt.Errorf("lock %s migrations: %w", r.config.Service, err)
	}
	return nil
}

func (r *Runner) lockName() string {
	return "purrchat:migrations:" + r.config.Service
}

func ensureMigrationTable(ctx context.Context, conn *pgxpool.Conn) error {
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS purrchat_schema_migrations (
		service TEXT NOT NULL,
		version TEXT NOT NULL,
		filename TEXT NOT NULL,
		checksum CHAR(64) NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (service, version),
		UNIQUE (service, filename)
	)`)
	if err != nil {
		return fmt.Errorf("create migration history table: %w", err)
	}
	return nil
}

func (r *Runner) verifyApplied(ctx context.Context, conn *pgxpool.Conn, migrations []Migration) (map[string]bool, error) {
	known := make(map[string]Migration, len(migrations))
	for _, migration := range migrations {
		known[migration.Version] = migration
	}

	rows, err := conn.Query(ctx,
		`SELECT version, filename, checksum FROM purrchat_schema_migrations WHERE service = $1 ORDER BY version`,
		r.config.Service)
	if err != nil {
		return nil, fmt.Errorf("read migration history: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool, len(migrations))
	for rows.Next() {
		var version, filename, checksum string
		if err := rows.Scan(&version, &filename, &checksum); err != nil {
			return nil, fmt.Errorf("scan migration history: %w", err)
		}
		migration, exists := known[version]
		if !exists {
			return nil, fmt.Errorf("applied %s migration %q is missing from source", r.config.Service, version)
		}
		if migration.Filename != filename {
			return nil, fmt.Errorf("applied migration %q was renamed from %q to %q", version, filename, migration.Filename)
		}
		if migration.Checksum != strings.TrimSpace(checksum) {
			return nil, fmt.Errorf("checksum changed for applied migration %q (%s)", version, filename)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration history: %w", err)
	}
	return applied, nil
}

func (r *Runner) baselineTableExists(ctx context.Context, conn *pgxpool.Conn) (bool, error) {
	if r.config.BaselineTable == "" {
		return false, fmt.Errorf("baseline table is required")
	}
	var exists bool
	err := conn.QueryRow(ctx, "SELECT to_regclass(format('public.%I', $1::text)) IS NOT NULL", r.config.BaselineTable).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check baseline table %q: %w", r.config.BaselineTable, err)
	}
	return exists, nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, service string, migration Migration) error {
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Filename, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", migration.Filename, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO purrchat_schema_migrations (service, version, filename, checksum)
		 VALUES ($1, $2, $3, $4)`, service, migration.Version, migration.Filename, migration.Checksum); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Filename, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Filename, err)
	}
	return nil
}
