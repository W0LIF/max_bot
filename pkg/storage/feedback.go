package storage

import (
	"context"
	"time"
)

type Feedback struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SQLiteStore) CreateFeedback(ctx context.Context, f *Feedback) (int64, error) {
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO feedback(user_id, text, created_at) VALUES(?, ?, ?)`,
		f.UserID, f.Text, f.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (m *MemoryStore) CreateFeedback(_ context.Context, f *Feedback) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}
	m.feedbackSeq++
	cp := *f
	cp.ID = m.feedbackSeq
	m.feedback = append(m.feedback, cp)
	return cp.ID, nil
}
