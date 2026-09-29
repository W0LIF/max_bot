package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	mu sync.Mutex
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// :memory: живёт в одном соединении — иначе таблицы из migrate()
	// не увидят последующие запросы.
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id           INTEGER PRIMARY KEY,
	name         TEXT NOT NULL DEFAULT '',
	consent      INTEGER NOT NULL DEFAULT 0,
	reminders_on INTEGER NOT NULL DEFAULT 0,
	onboarded_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id       INTEGER NOT NULL,
	title         TEXT NOT NULL,
	subject       TEXT NOT NULL DEFAULT '',
	deadline      TIMESTAMP,
	urgent        INTEGER NOT NULL DEFAULT 0,
	important     INTEGER NOT NULL DEFAULT 0,
	status        TEXT NOT NULL DEFAULT 'new',
	reminder_sent INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS moods (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL,
	value      TEXT NOT NULL,
	note       TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	text    TEXT NOT NULL,
	all_day INTEGER NOT NULL DEFAULT 0,
	start   TIMESTAMP NOT NULL,
	end     TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS feedback (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL,
	text       TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS group_members (
	user_id   INTEGER NOT NULL,
	group_id  INTEGER NOT NULL,
	joined_at TIMESTAMP NOT NULL,
	PRIMARY KEY (user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_deadline ON tasks(deadline);
CREATE INDEX IF NOT EXISTS idx_moods_user_id ON moods(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO groups(id, name) VALUES (1, 'ИКТн-54')`); err != nil {
		return err
	}

	// Условный ALTER для существующих БД.
	if err := s.ensureColumn("tasks", "subject", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ensureColumn(table, column, decl string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, decl))
	return err
}

// --- Задачи ---

func (s *SQLiteStore) CreateTask(_ context.Context, t *Task) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
		INSERT INTO tasks (user_id, title, subject, deadline, urgent, important, status, reminder_sent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.UserID,
		t.Title,
		t.Subject,
		t.Deadline,
		boolToInt(t.Urgent),
		boolToInt(t.Important),
		string(t.Status),
		boolToInt(t.ReminderSent),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetTasks(_ context.Context, userID int64) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, user_id, title, subject, deadline, urgent, important, status, reminder_sent
		FROM tasks WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, err
	}
	sortTasks(tasks)
	return tasks, nil
}

func (s *SQLiteStore) GetTask(_ context.Context, userID, taskID int64) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.getTaskLocked(userID, taskID)
}

func (s *SQLiteStore) UpdateTask(_ context.Context, t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
		UPDATE tasks SET
			title         = ?,
			subject       = ?,
			deadline      = ?,
			urgent        = ?,
			important     = ?,
			status        = ?,
			reminder_sent = ?
		WHERE id = ? AND user_id = ?
	`,
		t.Title,
		t.Subject,
		t.Deadline,
		boolToInt(t.Urgent),
		boolToInt(t.Important),
		string(t.Status),
		boolToInt(t.ReminderSent),
		t.ID,
		t.UserID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrTaskNotFound)
}

func (s *SQLiteStore) DeleteTask(_ context.Context, userID, taskID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		`DELETE FROM tasks WHERE id = ? AND user_id = ?`,
		taskID, userID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrTaskNotFound)
}

func (s *SQLiteStore) SetTaskStatus(_ context.Context, userID, taskID int64, status TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.getTaskLocked(userID, taskID)
	if err != nil {
		return err
	}
	if err := t.CanTransitionTo(status); err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE tasks SET status = ? WHERE id = ? AND user_id = ?`,
		string(status), taskID, userID,
	)
	return err
}

func (s *SQLiteStore) GetTasksDueBefore(_ context.Context, before time.Time) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, user_id, title, subject, deadline, urgent, important, status, reminder_sent
		FROM tasks
		WHERE deadline IS NOT NULL
		  AND deadline > '1900-01-01'
		  AND deadline < ?
		  AND status != ?
		  AND reminder_sent = 0
	`, before, string(TaskDone))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, err
	}
	sortTasks(tasks)
	return tasks, nil
}

func (s *SQLiteStore) SetTaskReminderSent(_ context.Context, taskID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		`UPDATE tasks SET reminder_sent = 1 WHERE id = ?`,
		taskID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrTaskNotFound)
}

// --- helpers ---

func (s *SQLiteStore) getTaskLocked(userID, taskID int64) (*Task, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, title, subject, deadline, urgent, important, status, reminder_sent
		FROM tasks WHERE id = ? AND user_id = ?
	`, taskID, userID)

	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func scanTask(row *sql.Row) (*Task, error) {
	var (
		t            Task
		urgent       int
		important    int
		status       string
		reminderSent int
	)
	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Subject, &t.Deadline,
		&urgent, &important, &status, &reminderSent,
	)
	if err != nil {
		return nil, err
	}
	t.Urgent = urgent != 0
	t.Important = important != 0
	t.Status = TaskStatus(status)
	t.ReminderSent = reminderSent != 0
	return &t, nil
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var result []Task
	for rows.Next() {
		var (
			t            Task
			urgent       int
			important    int
			status       string
			reminderSent int
		)
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Subject, &t.Deadline,
			&urgent, &important, &status, &reminderSent,
		)
		if err != nil {
			return nil, err
		}
		t.Urgent = urgent != 0
		t.Important = important != 0
		t.Status = TaskStatus(status)
		t.ReminderSent = reminderSent != 0
		result = append(result, t)
	}
	return result, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func checkRowsAffected(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

var _ Store = (*SQLiteStore)(nil)
var _ Store = (*MemoryStore)(nil)
