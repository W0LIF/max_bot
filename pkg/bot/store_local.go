package bot

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Временное хранилище в памяти пока нет  pkg/storage.

type TaskStatus int

const (
	TaskNew TaskStatus = iota
	TaskInProgress
	TaskDone
)

type User struct {
	ID          int64
	Consent     bool
	Reminders   bool
	OnboardedAt time.Time
}

type Task struct {
	ID       int64
	UserID   int64
	Title    string
	Deadline time.Time
	Priority int // 1..4
	Status   TaskStatus
}

type Store interface {
	GetUser(ctx context.Context, userID int64) (*User, error)
	UpsertUser(ctx context.Context, u *User) error
	SetConsent(ctx context.Context, userID int64, consent bool) error
	SetReminders(ctx context.Context, userID int64, enabled bool) error

	CreateTask(ctx context.Context, t *Task) (int64, error)
	GetTasks(ctx context.Context, userID int64) ([]Task, error)
	GetTask(ctx context.Context, userID, taskID int64) (*Task, error)
	UpdateTask(ctx context.Context, t *Task) error
	DeleteTask(ctx context.Context, userID, taskID int64) error
	SetTaskStatus(ctx context.Context, userID, taskID int64, status TaskStatus) error
}

var ErrNotFound = errors.New("не найдено")

type memStore struct {
	mu       sync.Mutex
	users    map[int64]*User
	tasks    map[int64]*Task
	nextTask int64
}

func NewMemStore() Store {
	return &memStore{
		users: make(map[int64]*User),
		tasks: make(map[int64]*Task),
	}
}

func (m *memStore) GetUser(_ context.Context, userID int64) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memStore) UpsertUser(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	if cp.OnboardedAt.IsZero() {
		cp.OnboardedAt = time.Now()
	}
	m.users[u.ID] = &cp
	return nil
}

func (m *memStore) SetConsent(_ context.Context, userID int64, consent bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		u = &User{ID: userID, OnboardedAt: time.Now()}
		m.users[userID] = u
	}
	u.Consent = consent
	return nil
}

func (m *memStore) SetReminders(_ context.Context, userID int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		u = &User{ID: userID, OnboardedAt: time.Now()}
		m.users[userID] = u
	}
	u.Reminders = enabled
	return nil
}

func (m *memStore) CreateTask(_ context.Context, t *Task) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextTask++
	cp := *t
	cp.ID = m.nextTask
	if cp.Status == 0 {
		cp.Status = TaskNew
	}
	m.tasks[cp.ID] = &cp
	return cp.ID, nil
}

func (m *memStore) GetTasks(_ context.Context, userID int64) ([]Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Task
	for _, t := range m.tasks {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (m *memStore) GetTask(_ context.Context, userID, taskID int64) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok || t.UserID != userID {
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *memStore) UpdateTask(_ context.Context, t *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	cp := *t
	m.tasks[t.ID] = &cp
	return nil
}

func (m *memStore) DeleteTask(_ context.Context, userID, taskID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok || t.UserID != userID {
		return ErrNotFound
	}
	delete(m.tasks, taskID)
	return nil
}

func (m *memStore) SetTaskStatus(_ context.Context, userID, taskID int64, status TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok || t.UserID != userID {
		return ErrNotFound
	}
	t.Status = status
	return nil
}
