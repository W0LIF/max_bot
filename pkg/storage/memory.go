package storage

import (
	"sort"
	"sync"
)

// MemoryStore — реализация хранилища «в памяти».
//
// Данные живут в обычных map и теряются при перезапуске процесса.
// Используется для разработки, тестов и как замена SQLite, пока тот
// не готов. Поведение методов должно совпадать с SQLiteStore —
// чтобы обе реализации можно было менять местами.
type MemoryStore struct {
	mu         sync.Mutex
	users      map[int64]*User
	tasks      map[int64]*Task
	nextTaskID int64
}

// NewMemoryStore создаёт пустое хранилище в памяти.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:      make(map[int64]*User),
		tasks:      make(map[int64]*Task),
		nextTaskID: 1,
	}
}

// --- Пользователь ---

// SaveUser создаёт или перезаписывает пользователя по его ID.
func (s *MemoryStore) SaveUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *u
	s.users[u.ID] = &cp
	return nil
}

// GetUser возвращает пользователя по ID или ErrUserNotFound.
func (s *MemoryStore) GetUser(id int64) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

// SetConsent меняет флаг согласия у пользователя.
func (s *MemoryStore) SetConsent(id int64, consent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Consent = consent
	return nil
}

// SetReminders меняет флаг напоминаний у пользователя.
func (s *MemoryStore) SetReminders(id int64, on bool) error {
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

// CreateTask сохраняет новую задачу и возвращает её ID.
func (s *MemoryStore) CreateTask(t *Task) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextTaskID
	s.nextTaskID++

	cp := *t
	cp.ID = id
	s.tasks[id] = &cp
	return id, nil
}

// GetTasks возвращает задачи пользователя, отсортированные:
// сначала важные, потом срочные, потом по дедлайну.
func (s *MemoryStore) GetTasks(userID int64) ([]Task, error) {
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

// GetTask возвращает задачу по ID или ErrTaskNotFound.
func (s *MemoryStore) GetTask(id int64) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	cp := *t
	return &cp, nil
}

// UpdateTask перезаписывает задачу по её ID.
func (s *MemoryStore) UpdateTask(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[t.ID]; !ok {
		return ErrTaskNotFound
	}
	cp := *t
	s.tasks[t.ID] = &cp
	return nil
}

// DeleteTask удаляет задачу по ID.
func (s *MemoryStore) DeleteTask(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrTaskNotFound
	}
	delete(s.tasks, id)
	return nil
}

// SetTaskStatus меняет статус задачи с проверкой перехода.
func (s *MemoryStore) SetTaskStatus(id int64, status TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	if err := t.CanTransitionTo(status); err != nil {
		return err
	}
	t.Status = status
	return nil
}

// sortTasks сортирует срез задач по правилу:
// Important=true идут раньше, потом Urgent=true, потом по Deadline.
//
// Функция общая для всех реализаций Store — чтобы порядок был одинаковым.
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
