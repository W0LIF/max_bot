package storage

import (
	"context"
	"testing"
)

func TestSQLiteStore(t *testing.T) {
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть БД: %v", err)
	}
	defer s.Close()

	runStoreTests(t, s)
}

func TestSQLiteStore_MoodsNotesFeedback(t *testing.T) {
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть БД: %v", err)
	}
	defer s.Close()

	runExtendedStoreTests(t, s)
}

func TestSQLiteStore_Groups(t *testing.T) {
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть БД: %v", err)
	}
	defer s.Close()

	runGroupStoreTests(t, s)
}

func TestNewConfiguredStore_UsesLocalSQLiteWithoutTurso(t *testing.T) {
	t.Setenv("TURSO_DATABASE_URL", "")
	t.Setenv("TURSO_AUTH_TOKEN", "")

	store, err := NewConfiguredStore(":memory:")
	if err != nil {
		t.Fatalf("NewConfiguredStore: %v", err)
	}
	defer store.Close()

	if err := store.SaveUser(context.Background(), &User{ID: 1, Name: "Локальный пользователь"}); err != nil {
		t.Fatalf("SaveUser: %v", err)
	}
}

func TestNewConfiguredStore_RequiresTursoToken(t *testing.T) {
	t.Setenv("TURSO_DATABASE_URL", "libsql://example.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "")

	if _, err := NewConfiguredStore(":memory:"); err == nil {
		t.Fatal("ожидали ошибку без TURSO_AUTH_TOKEN")
	}
}

func TestSQLiteStore_MigrateExistingDB(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/old.db"

	old, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("initial open: %v", err)
	}
	old.Close()

	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s.Close()

	if err := s.SaveUser(context.Background(), &User{ID: 1, Name: "Тест"}); err != nil {
		t.Fatalf("SaveUser: %v", err)
	}
	u, err := s.GetUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.Name != "Тест" {
		t.Fatalf("Name не сохранился: %q", u.Name)
	}
}
