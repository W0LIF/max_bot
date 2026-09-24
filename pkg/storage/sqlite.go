package storage

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	mu sync.Mutex
	db *sql.DB
}

// NewSQLiteStore открывает БД по пути path и применяет миграции.
//
// Если path == ":memory:" — БД живёт только в памяти процесса
// (удобно для тестов). Если path — путь к файлу, БД сохраняется.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// migrate создаёт таблицы, если их ещё нет.
func (s *SQLiteStore) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id           INTEGER PRIMARY KEY,
	consent      INTEGER NOT NULL DEFAULT 0,
	reminders_on INTEGER NOT NULL DEFAULT 0,
	onboarded_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL,
	title      TEXT NOT NULL,
	deadline   TIMESTAMP,
	urgent     INTEGER NOT NULL DEFAULT 0,
	important  INTEGER NOT NULL DEFAULT 0,
	status     TEXT NOT NULL DEFAULT 'new'
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
`

	_, err := s.db.Exec(schema)
	return err
}
