// Package storetest opens throwaway databases for tests.
package storetest

import (
	"path/filepath"
	"testing"

	"github.com/alias-asso/iosu/internal/store"
)

// New returns a migrated, empty database in a directory that is removed when
// the test ends.
func New(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
