package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
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

func TestSQLiteStore_MigratesLegacyGroupMembershipWithoutConsent(t *testing.T) {
	path := t.TempDir() + "/legacy.db"
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`CREATE TABLE group_members (
		user_id INTEGER NOT NULL,
		group_id INTEGER NOT NULL,
		joined_at TIMESTAMP NOT NULL,
		PRIMARY KEY (user_id, group_id)
	)`)
	if err != nil {
		legacy.Close()
		t.Fatal(err)
	}
	_, err = legacy.Exec(`INSERT INTO group_members(user_id, group_id, joined_at) VALUES(42, 1, ?)`, time.Now().UTC())
	if err != nil {
		legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("open migrated DB: %v", err)
	}
	defer store.Close()

	if _, err := store.GetUserGroup(context.Background(), 42); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("legacy membership must not count as consent: %v", err)
	}
	if err := store.EnsureGroup(context.Background(), 42, DemoGroupID); err != nil {
		t.Fatalf("explicit opt-in failed: %v", err)
	}
	if gid, err := store.GetUserGroup(context.Background(), 42); err != nil || gid != DemoGroupID {
		t.Fatalf("explicit opt-in not persisted: group=%d err=%v", gid, err)
	}
}
