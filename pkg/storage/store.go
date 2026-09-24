package storage

type Store interface {
	SaveUser(u *User) error

	GetUser(id int64) (*User, error)

	SetConsent(id int64) (*User, error)

	SetReminders(id int64, on bool) error

	CreateTask(t *Task) (int64, error)

	GetTasks(userID int64) ([]Task, error)

	GetTask(id int64) (*Task, error)

	UpdateTask(t *Task) error

	DeleteTask(id int64) error

	SetTaskStatus(id int64, status TaskStatus) error
}
