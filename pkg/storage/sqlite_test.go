package storage

import "testing"

func TestSQLiteStore(t *testing.T) {
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть БД: %v", err)
	}
	runStoreTests(t, s)
}
