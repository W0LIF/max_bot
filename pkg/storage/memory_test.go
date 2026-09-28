package storage

import "testing"

func TestMemoryStore(t *testing.T) {
	runStoreTests(t, NewMemoryStore())
}
