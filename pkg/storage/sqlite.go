package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

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

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

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

// --- Пользователь ---

func (s *SQLiteStore) SaveUser(_ context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO users (id, consent, reminders_on, onboarded_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			consent      = excluded.consent,
			reminders_on = excluded.reminders_on,
			onboarded_at = excluded.onboarded_at
	`,
		u.ID,
		boolToInt(u.Consent),
		boolToInt(u.RemindersOn),
		u.OnboardedAt,
	)
	return err
}

func (s *SQLiteStore) GetUser(_ context.Context, id int64) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRow(`
		SELECT id, consent, reminders_on, onboarded_at
		FROM users WHERE id = ?
	`, id)

	var (
		u           User
		consent     int
		remindersOn int
	)

	err := row.Scan(&u.ID, &consent, &remindersOn, &u.OnboardedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	u.Consent = consent != 0
	u.RemindersOn = remindersOn != 0
	return &u, nil
}

func (s *SQLiteStore) SetConsent(_ context.Context, id int64, consent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		`UPDATE users SET consent = ? WHERE id = ?`,
		boolToInt(consent), id,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrUserNotFound)
}

func (s *SQLiteStore) SetReminders(_ context.Context, id int64, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		`UPDATE users SET reminders_on = ? WHERE id = ?`,
		boolToInt(on), id,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrUserNotFound)
}

// --- Задачи ---

func (s *SQLiteStore) CreateTask(_ context.Context, t *Task) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
		INSERT INTO tasks (user_id, title, deadline, urgent, important, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		t.UserID,
		t.Title,
		t.Deadline,
		boolToInt(t.Urgent),
		boolToInt(t.Important),
		string(t.Status),
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *SQLiteStore) GetTasks(_ context.Context, userID int64) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, user_id, title, deadline, urgent, important, status
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
			user_id   = ?,
			title     = ?,
			deadline  = ?,
			urgent    = ?,
			important = ?,
			status    = ?
		WHERE id = ?
	`,
		t.UserID,
		t.Title,
		t.Deadline,
		boolToInt(t.Urgent),
		boolToInt(t.Important),
		string(t.Status),
		t.ID,
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
		`UPDATE tasks SET status = ? WHERE id = ?`,
		string(status), taskID,
	)
	return err
}

// --- Вспомогательные ---

func (s *SQLiteStore) getTaskLocked(userID, taskID int64) (*Task, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, title, deadline, urgent, important, status
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
		t         Task
		urgent    int
		important int
		status    string
	)

	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Deadline,
		&urgent, &important, &status,
	)
	if err != nil {
		return nil, err
	}

	t.Urgent = urgent != 0
	t.Important = important != 0
	t.Status = TaskStatus(status)
	return &t, nil
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var result []Task

	for rows.Next() {
		var (
			t         Task
			urgent    int
			important int
			status    string
		)

		err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Deadline,
			&urgent, &important, &status,
		)
		if err != nil {
			return nil, err
		}

		t.Urgent = urgent != 0
		t.Important = important != 0
		t.Status = TaskStatus(status)
		result = append(result, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
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
