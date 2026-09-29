package storage

import (
	"context"
	"time"
)

// Store — общий интерфейс хранилища.
// Реализуется SQLiteStore (прод) и MemoryStore (локальные тесты, webhook
// в dev-режиме).
//
// Все методы, работающие с задачами/настроениями/заметками, фильтруют
// по user_id. Исключение — SetTaskReminderSent и GetTasksDueBefore:
// они обслуживают шедулер напоминаний и работают по всем пользователям.
type Store interface {
	// Users
	SaveUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, id int64) (*User, error)
	SetConsent(ctx context.Context, id int64, consent bool) error
	SetReminders(ctx context.Context, id int64, on bool) error

	// Tasks
	CreateTask(ctx context.Context, t *Task) (int64, error)
	GetTasks(ctx context.Context, userID int64) ([]Task, error)
	GetTask(ctx context.Context, userID, taskID int64) (*Task, error)
	UpdateTask(ctx context.Context, t *Task) error
	DeleteTask(ctx context.Context, userID, taskID int64) error
	SetTaskStatus(ctx context.Context, userID, taskID int64, status TaskStatus) error

	// GetTasksDueBefore — все задачи всех пользователей с дедлайном
	// до before и ReminderSent=false. Для шедулера, не для API.
	GetTasksDueBefore(ctx context.Context, before time.Time) ([]Task, error)
	SetTaskReminderSent(ctx context.Context, taskID int64) error

	// Moods
	CreateMood(ctx context.Context, m *Mood) (int64, error)
	GetMoods(ctx context.Context, userID int64, days int) ([]Mood, error)

	// Notes
	CreateNote(ctx context.Context, n *Note) (int64, error)
	GetNotes(ctx context.Context, userID int64) ([]Note, error)
	DeleteNote(ctx context.Context, userID, id int64) error

	// Feedback
	CreateFeedback(ctx context.Context, f *Feedback) (int64, error)

	// Groups
	EnsureGroup(ctx context.Context, userID, groupID int64) error
	GetUserGroup(ctx context.Context, userID int64) (int64, error)
	GetGroupMembers(ctx context.Context, groupID int64) ([]Member, error)
}
