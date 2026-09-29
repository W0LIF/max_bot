package storage

import "testing"

func TestMemoryStore(t *testing.T) {
	runStoreTests(t, NewMemoryStore())
}

func TestMemoryStore_MoodsNotesFeedback(t *testing.T) {
	runExtendedStoreTests(t, NewMemoryStore())
}

func TestMemoryStore_Groups(t *testing.T) {
	runGroupStoreTests(t, NewMemoryStore())
}
