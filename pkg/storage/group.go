package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrGroupNotFound = errors.New("storage: группа не найдена")

const (
	DemoGroupID   int64 = 1
	DemoGroupName       = "ИКТн-54"
)

type Member struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
	Tasks int    `json:"tasks"`
}

func (s *SQLiteStore) EnsureGroup(ctx context.Context, userID, groupID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO group_members(user_id, group_id, joined_at)
		 VALUES(?, ?, ?)`, userID, groupID, time.Now().UTC())
	return err
}

func (s *SQLiteStore) GetUserGroup(ctx context.Context, userID int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var gid int64
	err := s.db.QueryRowContext(ctx,
		`SELECT group_id FROM group_members WHERE user_id = ?`, userID).Scan(&gid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrGroupNotFound
		}
		return 0, err
	}
	return gid, nil
}

func (s *SQLiteStore) GetGroupMembers(ctx context.Context, groupID int64) ([]Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.name, g.name,
		       (SELECT COUNT(*) FROM tasks t
		        WHERE t.user_id = u.id AND t.status != 'done') AS active
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		JOIN groups g ON g.id = gm.group_id
		WHERE gm.group_id = ?
		ORDER BY u.name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.Group, &m.Tasks); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (m *MemoryStore) EnsureGroup(_ context.Context, userID, groupID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.userGroups == nil {
		m.userGroups = map[int64]int64{}
	}
	if _, ok := m.userGroups[userID]; !ok {
		m.userGroups[userID] = groupID
	}
	return nil
}

func (m *MemoryStore) GetUserGroup(_ context.Context, userID int64) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gid, ok := m.userGroups[userID]
	if !ok {
		return 0, ErrGroupNotFound
	}
	return gid, nil
}

func (m *MemoryStore) GetGroupMembers(_ context.Context, groupID int64) ([]Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []Member
	for uid, gid := range m.userGroups {
		if gid != groupID {
			continue
		}
		name := "Студент"
		if u := m.users[uid]; u != nil && u.Name != "" {
			name = u.Name
		}
		active := 0
		for _, t := range m.tasks {
			if t.UserID == uid && t.Status != TaskDone {
				active++
			}
		}
		out = append(out, Member{ID: uid, Name: name, Group: DemoGroupName, Tasks: active})
	}
	return out, nil
}
