package storage

import (
	"context"
	"time"
)

type Store interface {
	SaveUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, id int64) (*User, error)
	SetConsent(ctx context.Context, id int64, consent bool) error
	SetReminders(ctx context.Context, id int64, on bool) error

	CreateTask(ctx context.Context, t *Task) (int64, error)
	GetTasks(ctx context.Context, userID int64) ([]Task, error)
	GetTask(ctx context.Context, userID, taskID int64) (*Task, error)
	UpdateTask(ctx context.Context, t *Task) error
	DeleteTask(ctx context.Context, userID, taskID int64) error
	SetTaskStatus(ctx context.Context, userID, taskID int64, status TaskStatus) error
	GetTasksDueBefore(ctx context.Context, before time.Time) ([]Task, error)
	SetTaskReminderSent(ctx context.Context, taskID int64) error
}
