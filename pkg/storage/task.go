package storage

import (
	"errors"
	"time"
)

type TaskStatus string

const (
	TaskNew        TaskStatus = "new"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskNew, TaskInProgress, TaskDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Title     string     `json:"title"`
	Deadline  time.Time  `json:"deadline"`
	Urgent    bool       `json:"urgent"`
	Important bool       `json:"important"`
	Status    TaskStatus `json:"status"`
}

var (
	ErrTaskNotFound      = errors.New("storage: задача не найдена")
	ErrInvalidStatus     = errors.New("storage: недопустимый статус")
	ErrInvalidTransition = errors.New("storage: недопустимый переход статуса")
	ErrUserNotFound      = errors.New("storage: пользователь не найден")
)

// CanTransitionTo проверяет, разрешён ли переход из текущего статуса
// задачи в next. Возвращает nil, если переход разрешён, и одну из
// ошибок (ErrInvalidStatus / ErrInvalidTransition) — если нет
func (t *Task) CanTransitionTo(next TaskStatus) error {
	if !next.IsValid() {
		return ErrInvalidStatus
	}
	if t.Status == next {
		return nil // идемпотентно
	}
	switch t.Status {
	case TaskNew:
		if next == TaskInProgress || next == TaskDone {
			return nil
		}
	case TaskInProgress:
		if next == TaskDone || next == TaskNew {
			return nil
		}
	case TaskDone:
		if next == TaskNew {
			return nil
		}
	}
	return ErrInvalidTransition
}
