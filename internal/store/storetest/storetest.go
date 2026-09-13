package storetest

import (
	"path/filepath"
	"testing"

	"github.com/alias-asso/iosu/internal/store"
)

func New(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
