package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateFromEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("reading version: %v", err)
	}
	if version == 0 {
		t.Fatal("user_version is still 0, migrations did not run")
	}
	s.Close()

	// Re-opening must be a no-op, not an error.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer s2.Close()

	var version2 int
	if err := s2.db.QueryRow("PRAGMA user_version").Scan(&version2); err != nil {
		t.Fatalf("reading version: %v", err)
	}
	if version2 != version {
		t.Fatalf("version changed on re-open: %d -> %d", version, version2)
	}
}

func TestOpenRejectsLegacyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// `configs` is GORM's site-config table and exists in no current migration.
	if _, err := s.db.Exec("CREATE TABLE configs (id integer)"); err != nil {
		t.Fatalf("seeding legacy table: %v", err)
	}
	s.Close()

	if _, err := Open(path); err == nil {
		t.Fatal("expected Open to refuse a GORM-shaped database")
	} else if !strings.Contains(err.Error(), "pre-sqlc") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrationNumber(t *testing.T) {
	for _, tc := range []struct {
		name    string
		want    int
		wantErr bool
	}{
		{name: "001_init.sql", want: 1},
		{name: "012_add_sessions.sql", want: 12},
		{name: "init.sql", wantErr: true},
		{name: "abc_init.sql", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := migrationNumber(tc.name)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "fk.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	_, err = s.db.Exec("INSERT INTO solves (user_id, problem_id, parts, solved_at) VALUES (1, 1, 1, 0)")
	if err == nil {
		t.Fatal("expected a foreign key violation")
	}
}
