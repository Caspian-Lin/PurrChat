package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRejectsDuplicateVersions(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_first.sql", "SELECT 1;")
	writeMigration(t, dir, "001_second.sql", "SELECT 2;")

	_, err := NewRunner(Config{Service: "backend", Dir: dir, BaselineTable: "users"}).Discover()
	if err == nil || !contains(err.Error(), "duplicate migration version") {
		t.Fatalf("expected duplicate version error, got %v", err)
	}
}

func TestDiscoverUsesExplicitLegacyVersionAliases(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "006_permissions.sql", "SELECT 1;")
	writeMigration(t, dir, "006_ordering.sql", "SELECT 2;")

	migrations, err := NewRunner(Config{
		Service:       "backend",
		Dir:           dir,
		BaselineTable: "users",
		LegacyVersions: map[string]string{
			"006_permissions.sql": "006a",
			"006_ordering.sql":    "006b",
		},
	}).Discover()
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{migrations[0].Version, migrations[1].Version}; got[0] != "006a" || got[1] != "006b" {
		t.Fatalf("unexpected versions: %v", got)
	}
	if migrations[0].Checksum == migrations[1].Checksum {
		t.Fatal("expected distinct checksums")
	}
}

func TestDiscoverRejectsMissingLegacyFile(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_first.sql", "SELECT 1;")

	_, err := NewRunner(Config{
		Service:        "backend",
		Dir:            dir,
		BaselineTable:  "users",
		LegacyVersions: map[string]string{"006_missing.sql": "006a"},
	}).Discover()
	if err == nil || !contains(err.Error(), "references missing file") {
		t.Fatalf("expected missing legacy file error, got %v", err)
	}
}

func writeMigration(t *testing.T, dir, filename, sql string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}
}

func contains(value, part string) bool {
	return strings.Contains(value, part)
}
