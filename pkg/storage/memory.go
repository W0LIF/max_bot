package storage

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu sync.RWMutex

	users    map[int64]*User
	tasks    map[int64]*Task
	moods    []Mood
	notes    []Note
	feedback []Feedback

	nextTaskID  int64
	moodSeq     int64
	noteSeq     int64
	feedbackSeq int64

	userGroups map[int64]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:      make(map[int64]*User),
		tasks:      make(map[int64]*Task),
		nextTaskID: 1,
		userGroups: map[int64]int64{},
	}
}

// --- Задачи ---

func (s *MemoryStore) CreateTask(_ context.Context, t *Task) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextTaskID
	s.nextTaskID++

	cp := *t
	cp.ID = id
	if cp.Status == "" {
		cp.Status = TaskNew
	}
	s.tasks[id] = &cp
	return id, nil
}

func (s *MemoryStore) GetTasks(_ context.Context, userID int64) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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

	existing, ok := s.tasks[t.ID]
	if !ok || existing.UserID != t.UserID {
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

func (s *MemoryStore) GetTasksDueBefore(_ context.Context, before time.Time) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Task
	for _, t := range s.tasks {
		if t.Status == TaskDone || t.ReminderSent {
			continue
		}
		if t.Deadline.IsZero() {
			continue
		}
		if t.Deadline.Before(before) {
			result = append(result, *t)
		}
	}
	sortTasks(result)
	return result, nil
}

func (s *MemoryStore) SetTaskReminderSent(_ context.Context, taskID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if t.ReminderSent {
		return ErrReminderAlreadySent
	}
	t.ReminderSent = true
	return nil
}

// sortTasks — сначала Important, потом Urgent, потом по Deadline.
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
