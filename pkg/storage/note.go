package storage

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"
)

var ErrNoteNotFound = errors.New("storage: заметка не найдена")

type Note struct {
	ID     int64     `json:"id"`
	UserID int64     `json:"user_id"`
	Text   string    `json:"text"`
	AllDay bool      `json:"allDay"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

// --- SQLiteStore ---

func (s *SQLiteStore) CreateNote(ctx context.Context, n *Note) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO notes(user_id, text, all_day, start, end) VALUES(?, ?, ?, ?, ?)`,
		n.UserID, n.Text, boolToInt(n.AllDay), n.Start, n.End)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetNotes(ctx context.Context, userID int64) ([]Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, text, all_day, start, end
		 FROM notes WHERE user_id = ? ORDER BY start`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Note
	for rows.Next() {
		var (
			n      Note
			allDay int
		)
		if err := rows.Scan(&n.ID, &n.UserID, &n.Text, &allDay, &n.Start, &n.End); err != nil {
			return nil, err
		}
		n.AllDay = allDay != 0
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) DeleteNote(ctx context.Context, userID, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx,
		`DELETE FROM notes WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, ErrNoteNotFound)
}

// --- MemoryStore ---

func (m *MemoryStore) CreateNote(_ context.Context, n *Note) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.noteSeq++
	cp := *n
	cp.ID = m.noteSeq
	m.notes = append(m.notes, cp)
	return cp.ID, nil
}

func (m *MemoryStore) GetNotes(_ context.Context, userID int64) ([]Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []Note
	for _, n := range m.notes {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Start.Before(out[j].Start)
	})
	return out, nil
}

func (m *MemoryStore) DeleteNote(_ context.Context, userID, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, n := range m.notes {
		if n.ID == id && n.UserID == userID {
			m.notes = append(m.notes[:i], m.notes[i+1:]...)
			return nil
		}
	}
	return ErrNoteNotFound
}

// чтобы компилятор не ругался на неиспользуемый импорт, если уберёшь что-то
var _ = sql.ErrNoRows
