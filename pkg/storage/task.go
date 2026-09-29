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

// Task — задача пользователя.
// Subject — колонка B3. Group не сохраняется в БД, ставится вручную
// для GET /api/group/tasks.
type Task struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Title        string     `json:"title"`
	Subject      string     `json:"subject"`
	Deadline     time.Time  `json:"deadline"`
	Urgent       bool       `json:"urgent"`
	Important    bool       `json:"important"`
	Status       TaskStatus `json:"status"`
	ReminderSent bool       `json:"reminder_sent,omitempty"`

	Group bool `json:"group,omitempty"`
}

var (
	ErrTaskNotFound        = errors.New("storage: задача не найдена")
	ErrInvalidStatus       = errors.New("storage: недопустимый статус")
	ErrInvalidTransition   = errors.New("storage: недопустимый переход статуса")
	ErrReminderAlreadySent = errors.New("storage: напоминание уже отправлено")
)

func (t *Task) CanTransitionTo(next TaskStatus) error {
	if !next.IsValid() {
		return ErrInvalidStatus
	}
	if t.Status == next {
		return nil
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
