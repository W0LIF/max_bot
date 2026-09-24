package storage

import (
	"errors"
	"testing"
	"time"
)

// --- Пользователи ---

func TestGetUser_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetUser(999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ожидали ErrUserNotFound, получили %v", err)
	}
}

func TestSaveAndGetUser(t *testing.T) {
	s := NewMemoryStore()
	u := &User{ID: 1, Consent: true, RemindersOn: true, OnboardedAt: time.Now()}
	if err := s.SaveUser(u); err != nil {
		t.Fatalf("SaveUser вернул ошибку: %v", err)
	}

	got, err := s.GetUser(1)
	if err != nil {
		t.Fatalf("GetUser вернул ошибку: %v", err)
	}
	if got.ID != 1 || !got.Consent || !got.RemindersOn {
		t.Fatalf("данные пользователя не совпадают: %+v", got)
	}
}

func TestSetConsent(t *testing.T) {
	s := NewMemoryStore()
	_ = s.SaveUser(&User{ID: 1})

	if err := s.SetConsent(1, true); err != nil {
		t.Fatalf("SetConsent вернул ошибку: %v", err)
	}
	u, _ := s.GetUser(1)
	if !u.Consent {
		t.Fatalf("Consent не стал true")
	}
}

func TestSetConsent_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.SetConsent(999, true)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ожидали ErrUserNotFound, получили %v", err)
	}
}

func TestSetReminders(t *testing.T) {
	s := NewMemoryStore()
	_ = s.SaveUser(&User{ID: 1})

	if err := s.SetReminders(1, true); err != nil {
		t.Fatalf("SetReminders вернул ошибку: %v", err)
	}
	u, _ := s.GetUser(1)
	if !u.RemindersOn {
		t.Fatalf("RemindersOn не стал true")
	}
}

// --- Задачи ---

func TestCreateAndGetTask(t *testing.T) {
	s := NewMemoryStore()
	id, err := s.CreateTask(&Task{
		UserID:   1,
		Title:    "Сдать лабу",
		Deadline: time.Now().Add(24 * time.Hour),
		Status:   TaskNew,
	})
	if err != nil {
		t.Fatalf("CreateTask вернул ошибку: %v", err)
	}
	if id == 0 {
		t.Fatalf("CreateTask вернул нулевой ID")
	}

	got, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("GetTask вернул ошибку: %v", err)
	}
	if got.ID != id || got.Title != "Сдать лабу" {
		t.Fatalf("данные задачи не совпадают: %+v", got)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetTask(999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
	}
}

func TestGetTasks_OnlyOwnTasks(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.CreateTask(&Task{UserID: 1, Title: "A", Status: TaskNew})
	_, _ = s.CreateTask(&Task{UserID: 1, Title: "B", Status: TaskNew})
	_, _ = s.CreateTask(&Task{UserID: 2, Title: "C", Status: TaskNew})

	tasks, err := s.GetTasks(1)
	if err != nil {
		t.Fatalf("GetTasks вернул ошибку: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("ожидали 2 задачи, получили %d", len(tasks))
	}
	for _, task := range tasks {
		if task.UserID != 1 {
			t.Fatalf("вернулась чужая задача: %+v", task)
		}
	}
}

func TestGetTasks_Sorting(t *testing.T) {
	s := NewMemoryStore()
	now := time.Now()

	_, _ = s.CreateTask(&Task{UserID: 1, Title: "обычная", Deadline: now.Add(1 * time.Hour), Status: TaskNew})
	_, _ = s.CreateTask(&Task{UserID: 1, Title: "важная", Important: true, Deadline: now.Add(10 * time.Hour), Status: TaskNew})
	_, _ = s.CreateTask(&Task{UserID: 1, Title: "срочная", Urgent: true, Deadline: now.Add(5 * time.Hour), Status: TaskNew})
	_, _ = s.CreateTask(&Task{UserID: 1, Title: "важная и срочная", Important: true, Urgent: true, Deadline: now.Add(20 * time.Hour), Status: TaskNew})

	tasks, _ := s.GetTasks(1)
	want := []string{"важная и срочная", "важная", "срочная", "обычная"}
	for i, w := range want {
		if tasks[i].Title != w {
			t.Fatalf("позиция %d: ожидали %q, получили %q", i, w, tasks[i].Title)
		}
	}
}

func TestUpdateTask(t *testing.T) {
	s := NewMemoryStore()
	id, _ := s.CreateTask(&Task{UserID: 1, Title: "старое", Status: TaskNew})

	got, _ := s.GetTask(id)
	got.Title = "новое"
	if err := s.UpdateTask(got); err != nil {
		t.Fatalf("UpdateTask вернул ошибку: %v", err)
	}

	got2, _ := s.GetTask(id)
	if got2.Title != "новое" {
		t.Fatalf("название не обновилось: %q", got2.Title)
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateTask(&Task{ID: 999, Title: "нет такой"})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	s := NewMemoryStore()
	id, _ := s.CreateTask(&Task{UserID: 1, Title: "удалить", Status: TaskNew})

	if err := s.DeleteTask(id); err != nil {
		t.Fatalf("DeleteTask вернул ошибку: %v", err)
	}
	if _, err := s.GetTask(id); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("задача не удалилась")
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.DeleteTask(999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
	}
}

func TestSetTaskStatus_ValidTransition(t *testing.T) {
	s := NewMemoryStore()
	id, _ := s.CreateTask(&Task{UserID: 1, Title: "A", Status: TaskNew})

	if err := s.SetTaskStatus(id, TaskInProgress); err != nil {
		t.Fatalf("переход new → in_progress должен быть разрешён: %v", err)
	}
	if err := s.SetTaskStatus(id, TaskDone); err != nil {
		t.Fatalf("переход in_progress → done должен быть разрешён: %v", err)
	}
}

func TestSetTaskStatus_InvalidTransition(t *testing.T) {
	s := NewMemoryStore()
	id, _ := s.CreateTask(&Task{UserID: 1, Title: "A", Status: TaskDone})

	err := s.SetTaskStatus(id, TaskInProgress)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("ожидали ErrInvalidTransition, получили %v", err)
	}
}

func TestSetTaskStatus_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.SetTaskStatus(999, TaskDone)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
	}
}

// --- Изоляция копий ---

func TestGetUser_ReturnsCopy(t *testing.T) {
	s := NewMemoryStore()
	_ = s.SaveUser(&User{ID: 1, Consent: true})

	u, _ := s.GetUser(1)
	u.Consent = false

	u2, _ := s.GetUser(1)
	if !u2.Consent {
		t.Fatalf("изменение копии повлияло на хранилище")
	}
}
