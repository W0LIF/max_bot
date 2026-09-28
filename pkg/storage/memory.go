package storage

import (
	"context"
	"sort"
	"sync"
)

type MemoryStore struct {
	mu         sync.Mutex
	users      map[int64]*User
	tasks      map[int64]*Task
	nextTaskID int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:      make(map[int64]*User),
		tasks:      make(map[int64]*Task),
		nextTaskID: 1,
	}
}

// --- Пользователь ---

func (s *MemoryStore) SaveUser(_ context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *u
	s.users[u.ID] = &cp
	return nil
}

func (s *MemoryStore) GetUser(_ context.Context, id int64) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *MemoryStore) SetConsent(_ context.Context, id int64, consent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Consent = consent
	return nil
}

func (s *MemoryStore) SetReminders(_ context.Context, id int64, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.RemindersOn = on
	return nil
}

// --- Задачи ---

func (s *MemoryStore) CreateTask(_ context.Context, t *Task) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextTaskID
	s.nextTaskID++

	cp := *t
	cp.ID = id
	s.tasks[id] = &cp
	return id, nil
}

func (s *MemoryStore) GetTasks(_ context.Context, userID int64) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []Task
	for _, t := range s.tasks {
		if t.UserID == userID {
			result = append(result, *t)
		}
	}
	sortTasks(result)
	return result, nil
}

func (s *MemoryStore) GetTask(_ context.Context, userID, taskID int64) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok || t.UserID != userID {
		return nil, ErrTaskNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) UpdateTask(_ context.Context, t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[t.ID]; !ok {
		return ErrTaskNotFound
	}
	cp := *t
	s.tasks[t.ID] = &cp
	return nil
}

func (s *MemoryStore) DeleteTask(_ context.Context, userID, taskID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok || t.UserID != userID {
		return ErrTaskNotFound
	}
	delete(s.tasks, taskID)
	return nil
}

func (s *MemoryStore) SetTaskStatus(_ context.Context, userID, taskID int64, status TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok || t.UserID != userID {
		return ErrTaskNotFound
	}
	if err := t.CanTransitionTo(status); err != nil {
		return err
	}
	t.Status = status
	return nil
}

// sortTasks сортирует срез задач:
// сначала Important=true, потом Urgent=true, потом по Deadline.
func sortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.Important != b.Important {
			return a.Important
		}
		if a.Urgent != b.Urgent {
			return a.Urgent
		}
		return a.Deadline.Before(b.Deadline)
	})
}
