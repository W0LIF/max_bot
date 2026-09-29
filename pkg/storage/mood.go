package storage

import (
	"context"
	"sort"
	"time"
)

type MoodValue string

const (
	MoodGood MoodValue = "good"
	MoodOK   MoodValue = "ok"
	MoodBad  MoodValue = "bad"
)

func (v MoodValue) IsValid() bool {
	switch v {
	case MoodGood, MoodOK, MoodBad:
		return true
	default:
		return false
	}
}

type Mood struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Value     MoodValue `json:"value"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SQLiteStore) CreateMood(ctx context.Context, m *Mood) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO moods(user_id, value, note, created_at) VALUES(?, ?, ?, ?)`,
		m.UserID, m.Value, m.Note, m.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetMoods(ctx context.Context, userID int64, days int) ([]Mood, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	since := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, value, note, created_at
		 FROM moods WHERE user_id = ? AND created_at >= ?
		 ORDER BY created_at DESC`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Mood
	for rows.Next() {
		var m Mood
		if err := rows.Scan(&m.ID, &m.UserID, &m.Value, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (m *MemoryStore) CreateMood(_ context.Context, mood *Mood) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mood.CreatedAt.IsZero() {
		mood.CreatedAt = time.Now().UTC()
	}
	m.moodSeq++
	cp := *mood
	cp.ID = m.moodSeq
	m.moods = append(m.moods, cp)
	return cp.ID, nil
}

func (m *MemoryStore) GetMoods(_ context.Context, userID int64, days int) ([]Mood, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	since := time.Now().UTC().AddDate(0, 0, -days)
	var out []Mood
	for _, mood := range m.moods {
		if mood.UserID == userID && !mood.CreatedAt.Before(since) {
			out = append(out, mood)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
