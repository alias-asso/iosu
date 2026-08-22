// Package store owns the database: connection, schema migrations and the
// sqlc-generated queries. Everything above it talks to *Store, never to
// database/sql directly.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/alias-asso/iosu/internal/store/sqlc"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Store is the only handle on the database. The embedded *sqlc.Queries makes
// every generated query available directly on it.
type Store struct {
	*sqlc.Queries
	db *sql.DB
}

// Open connects to the SQLite database at path and brings it up to the latest
// schema version.
func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	// SQLite allows a single writer. Serialising every connection removes any
	// possibility of SQLITE_BUSY at the cost of throughput we do not need.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to %s: %w", path, err)
	}

	s := &Store{Queries: sqlc.New(db), db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Tx runs fn inside a transaction, rolling back if it returns an error.
func (s *Store) Tx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(s.Queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// migrate applies every embedded migration whose number is above the
// database's current PRAGMA user_version, each in its own transaction.
func (s *Store) migrate() error {
	if err := s.rejectLegacySchema(); err != nil {
		return err
	}

	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		n, err := migrationNumber(name)
		if err != nil {
			return err
		}
		if n <= version {
			continue
		}
		body, err := migrations.ReadFile(path.Join("migrations", name))
		if err != nil {
			return err
		}
		if err := s.applyMigration(n, string(body)); err != nil {
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) applyMigration(n int, body string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(body); err != nil {
		return err
	}
	// PRAGMA does not accept a bound parameter.
	if _, err := tx.Exec("PRAGMA user_version = " + strconv.Itoa(n)); err != nil {
		return err
	}
	return tx.Commit()
}

// migrationNumber parses the leading number of a "001_name.sql" filename.
func migrationNumber(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %q is not named NNN_description.sql", name)
	}
	n, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %q is not named NNN_description.sql", name)
	}
	return n, nil
}

// rejectLegacySchema refuses to touch a database left behind by the GORM
// version, whose tables differ in shape and would be silently mangled.
func (s *Store) rejectLegacySchema() error {
	var name string
	err := s.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'configs'`,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("this database was created by the pre-sqlc version of iosu; " +
		"move it aside and let iosud create a fresh one")
}
