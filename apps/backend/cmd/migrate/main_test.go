package main

import (
	"testing"

	migrate "github.com/Caspian-Lin/PurrChat/packages/db-migrate"
	"github.com/stretchr/testify/require"
)

func TestBackendLegacyMigrationsHaveStableLogicalVersions(t *testing.T) {
	runner := migrate.NewRunner(migrate.Config{
		Service:        "backend",
		Dir:            "../../migrations",
		BaselineTable:  "users",
		LegacyVersions: legacyVersions,
	})
	migrations, err := runner.Discover()
	require.NoError(t, err)

	versions := make(map[string]bool, len(migrations))
	for _, migration := range migrations {
		versions[migration.Version] = true
	}
	require.True(t, versions["006a"])
	require.True(t, versions["006b"])
	require.True(t, versions["011"])
	require.Len(t, versions, len(migrations))
}
