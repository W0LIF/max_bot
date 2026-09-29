package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("storage: пользователь не найден")

type User struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Consent     bool      `json:"consent"`
	RemindersOn bool      `json:"reminders_on"`
	OnboardedAt time.Time `json:"-"`
}

func (u *User) IsOnboarded() bool {
	return !u.OnboardedAt.IsZero()
}

// --- SQLiteStore ---

func (s *SQLiteStore) GetUser(ctx context.Context, id int64) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, consent, reminders_on, onboarded_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *SQLiteStore) SaveUser(ctx context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var onboarded any
	if !u.OnboardedAt.IsZero() {
		onboarded = u.OnboardedAt
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users(id, name, consent, reminders_on, onboarded_at)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name = excluded.name,
		   consent = excluded.consent,
		   reminders_on = excluded.reminders_on,
		   onboarded_at = excluded.onboarded_at`,
		u.ID, u.Name, boolToInt(u.Consent), boolToInt(u.RemindersOn), onboarded)
	return err
}

func (s *SQLiteStore) SetConsent(ctx context.Context, id int64, consent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET consent = ? WHERE id = ?`, boolToInt(consent), id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrUserNotFound)
}

func (s *SQLiteStore) SetReminders(ctx context.Context, id int64, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET reminders_on = ? WHERE id = ?`, boolToInt(on), id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrUserNotFound)
}

// --- MemoryStore ---

func (m *MemoryStore) GetUser(_ context.Context, id int64) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *MemoryStore) SaveUser(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users[u.ID] = &cp
	return nil
}

func (m *MemoryStore) SetConsent(_ context.Context, id int64, consent bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Consent = consent
	return nil
}

func (m *MemoryStore) SetReminders(_ context.Context, id int64, on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.RemindersOn = on
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(r rowScanner) (*User, error) {
	var u User
	var onboardedAt sql.NullTime
	if err := r.Scan(&u.ID, &u.Name, &u.Consent, &u.RemindersOn, &onboardedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if onboardedAt.Valid {
		u.OnboardedAt = onboardedAt.Time
	}
	return &u, nil
}
