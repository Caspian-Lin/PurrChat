package tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	migrate "github.com/Caspian-Lin/PurrChat/packages/db-migrate"
	"purr-chat-server/pkg/database"

	"github.com/stretchr/testify/require"
)

func TestMigrationRunnerAppliesOnceAndRejectsChecksumDrift(t *testing.T) {
	setupMigrationRunnerDB(t)
	ctx := context.Background()
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create_probe.sql", `
CREATE TABLE migration_runner_probe (id INTEGER PRIMARY KEY);
INSERT INTO migration_runner_probe (id) VALUES (1);
`)

	runner := migrate.NewRunner(migrate.Config{
		Service:       "backend-runner-test",
		Dir:           dir,
		BaselineTable: "migration_runner_probe",
	})
	require.NoError(t, runner.Run(ctx, database.GetPool()))
	require.NoError(t, runner.Run(ctx, database.GetPool()))

	var rows, applied int
	require.NoError(t, database.GetPool().QueryRow(ctx, `SELECT COUNT(*) FROM migration_runner_probe`).Scan(&rows))
	require.NoError(t, database.GetPool().QueryRow(ctx,
		`SELECT COUNT(*) FROM purrchat_schema_migrations WHERE service = 'backend-runner-test'`).Scan(&applied))
	require.Equal(t, 1, rows)
	require.Equal(t, 1, applied)

	writeMigrationFile(t, dir, "001_create_probe.sql", `
CREATE TABLE migration_runner_probe (id INTEGER PRIMARY KEY);
INSERT INTO migration_runner_probe (id) VALUES (2);
`)
	err := runner.Run(ctx, database.GetPool())
	require.ErrorContains(t, err, "checksum changed")
}

func TestMigrationRunnerRequiresExplicitBaseline(t *testing.T) {
	setupMigrationRunnerDB(t)
	ctx := context.Background()
	_, err := database.GetPool().Exec(ctx, `CREATE TABLE legacy_runner_probe (id INTEGER PRIMARY KEY)`)
	require.NoError(t, err)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_add_legacy_column.sql", `ALTER TABLE legacy_runner_probe ADD COLUMN migrated BOOLEAN NOT NULL DEFAULT false;`)

	runner := migrate.NewRunner(migrate.Config{
		Service:       "legacy-runner-test",
		Dir:           dir,
		BaselineTable: "legacy_runner_probe",
	})
	err = runner.Run(ctx, database.GetPool())
	require.ErrorContains(t, err, "without migration history")
	require.NoError(t, runner.Baseline(ctx, database.GetPool()))

	var applied int
	require.NoError(t, database.GetPool().QueryRow(ctx,
		`SELECT COUNT(*) FROM purrchat_schema_migrations WHERE service = 'legacy-runner-test'`).Scan(&applied))
	require.Equal(t, 1, applied)

	var migratedColumn bool
	require.NoError(t, database.GetPool().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'legacy_runner_probe' AND column_name = 'migrated'
)`).Scan(&migratedColumn))
	require.False(t, migratedColumn)
}

func TestMigrationRunnerSerializesConcurrentRuns(t *testing.T) {
	setupMigrationRunnerDB(t)
	ctx := context.Background()
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_wait.sql", `SELECT pg_sleep(0.2);`)

	runner := migrate.NewRunner(migrate.Config{
		Service:       "concurrent-runner-test",
		Dir:           dir,
		BaselineTable: "concurrent_runner_probe",
	})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- runner.Run(ctx, database.GetPool())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	var applied int
	require.NoError(t, database.GetPool().QueryRow(ctx,
		`SELECT COUNT(*) FROM purrchat_schema_migrations WHERE service = 'concurrent-runner-test'`).Scan(&applied))
	require.Equal(t, 1, applied)
}

func setupMigrationRunnerDB(t *testing.T) {
	t.Helper()
	SetupTestDB(t)
	t.Cleanup(func() { CleanupTestDB(t) })
	for _, table := range []string{"purrchat_schema_migrations", "migration_runner_probe", "legacy_runner_probe"} {
		_, err := database.GetPool().Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
		require.NoError(t, err)
	}
}

func writeMigrationFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o600))
}
