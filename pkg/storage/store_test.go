package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func runStoreTests(t *testing.T, s Store) {
	t.Helper()

	ctx := context.Background()

	// --- Пользователи ---

	t.Run("GetUser_NotFound", func(t *testing.T) {
		_, err := s.GetUser(ctx, 999)
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("ожидали ErrUserNotFound, получили %v", err)
		}
	})

	t.Run("SaveAndGetUser", func(t *testing.T) {
		u := &User{ID: 1, Consent: true, RemindersOn: true, OnboardedAt: time.Now()}
		if err := s.SaveUser(ctx, u); err != nil {
			t.Fatalf("SaveUser вернул ошибку: %v", err)
		}

		got, err := s.GetUser(ctx, 1)
		if err != nil {
			t.Fatalf("GetUser вернул ошибку: %v", err)
		}
		if got.ID != 1 || !got.Consent || !got.RemindersOn {
			t.Fatalf("данные пользователя не совпадают: %+v", got)
		}
	})

	t.Run("SetConsent", func(t *testing.T) {
		_ = s.SaveUser(ctx, &User{ID: 2})

		if err := s.SetConsent(ctx, 2, true); err != nil {
			t.Fatalf("SetConsent вернул ошибку: %v", err)
		}
		u, _ := s.GetUser(ctx, 2)
		if !u.Consent {
			t.Fatalf("Consent не стал true")
		}
	})

	t.Run("SetConsent_NotFound", func(t *testing.T) {
		err := s.SetConsent(ctx, 999, true)
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("ожидали ErrUserNotFound, получили %v", err)
		}
	})

	t.Run("SetReminders", func(t *testing.T) {
		_ = s.SaveUser(ctx, &User{ID: 3})

		if err := s.SetReminders(ctx, 3, true); err != nil {
			t.Fatalf("SetReminders вернул ошибку: %v", err)
		}
		u, _ := s.GetUser(ctx, 3)
		if !u.RemindersOn {
			t.Fatalf("RemindersOn не стал true")
		}
	})

	// --- Задачи ---

	t.Run("CreateAndGetTask", func(t *testing.T) {
		id, err := s.CreateTask(ctx, &Task{
			UserID:   10,
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

		got, err := s.GetTask(ctx, 10, id)
		if err != nil {
			t.Fatalf("GetTask вернул ошибку: %v", err)
		}
		if got.ID != id || got.Title != "Сдать лабу" {
			t.Fatalf("данные задачи не совпадают: %+v", got)
		}
	})

	t.Run("GetTask_NotFound", func(t *testing.T) {
		_, err := s.GetTask(ctx, 10, 999)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
		}
	})

	t.Run("GetTask_WrongUser", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 11, Title: "чужая", Status: TaskNew})

		_, err := s.GetTask(ctx, 12, id)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound для чужой задачи, получили %v", err)
		}
	})

	t.Run("GetTasks_OnlyOwnTasks", func(t *testing.T) {
		_, _ = s.CreateTask(ctx, &Task{UserID: 20, Title: "A", Status: TaskNew})
		_, _ = s.CreateTask(ctx, &Task{UserID: 20, Title: "B", Status: TaskNew})
		_, _ = s.CreateTask(ctx, &Task{UserID: 21, Title: "C", Status: TaskNew})

		tasks, err := s.GetTasks(ctx, 20)
		if err != nil {
			t.Fatalf("GetTasks вернул ошибку: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("ожидали 2 задачи, получили %d", len(tasks))
		}
		for _, task := range tasks {
			if task.UserID != 20 {
				t.Fatalf("вернулась чужая задача: %+v", task)
			}
		}
	})

	t.Run("GetTasks_Sorting", func(t *testing.T) {
		now := time.Now()

		_, _ = s.CreateTask(ctx, &Task{UserID: 30, Title: "обычная", Deadline: now.Add(1 * time.Hour), Status: TaskNew})
		_, _ = s.CreateTask(ctx, &Task{UserID: 30, Title: "важная", Important: true, Deadline: now.Add(10 * time.Hour), Status: TaskNew})
		_, _ = s.CreateTask(ctx, &Task{UserID: 30, Title: "срочная", Urgent: true, Deadline: now.Add(5 * time.Hour), Status: TaskNew})
		_, _ = s.CreateTask(ctx, &Task{UserID: 30, Title: "важная и срочная", Important: true, Urgent: true, Deadline: now.Add(20 * time.Hour), Status: TaskNew})

		tasks, _ := s.GetTasks(ctx, 30)
		want := []string{"важная и срочная", "важная", "срочная", "обычная"}
		for i, w := range want {
			if tasks[i].Title != w {
				t.Fatalf("позиция %d: ожидали %q, получили %q", i, w, tasks[i].Title)
			}
		}
	})

	t.Run("UpdateTask", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 40, Title: "старое", Status: TaskNew})

		got, _ := s.GetTask(ctx, 40, id)
		got.Title = "новое"
		if err := s.UpdateTask(ctx, got); err != nil {
			t.Fatalf("UpdateTask вернул ошибку: %v", err)
		}

		got2, _ := s.GetTask(ctx, 40, id)
		if got2.Title != "новое" {
			t.Fatalf("название не обновилось: %q", got2.Title)
		}
	})

	t.Run("UpdateTask_NotFound", func(t *testing.T) {
		err := s.UpdateTask(ctx, &Task{ID: 999, Title: "нет такой"})
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 50, Title: "удалить", Status: TaskNew})

		if err := s.DeleteTask(ctx, 50, id); err != nil {
			t.Fatalf("DeleteTask вернул ошибку: %v", err)
		}
		if _, err := s.GetTask(ctx, 50, id); !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("задача не удалилась")
		}
	})

	t.Run("DeleteTask_NotFound", func(t *testing.T) {
		err := s.DeleteTask(ctx, 50, 999)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
		}
	})

	t.Run("DeleteTask_WrongUser", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 51, Title: "не трогать", Status: TaskNew})

		err := s.DeleteTask(ctx, 52, id)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
		}

		if _, err := s.GetTask(ctx, 51, id); err != nil {
			t.Fatalf("задача была удалена чужим пользователем")
		}
	})

	t.Run("SetTaskStatus_ValidTransition", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 60, Title: "A", Status: TaskNew})

		if err := s.SetTaskStatus(ctx, 60, id, TaskInProgress); err != nil {
			t.Fatalf("переход new → in_progress должен быть разрешён: %v", err)
		}
		if err := s.SetTaskStatus(ctx, 60, id, TaskDone); err != nil {
			t.Fatalf("переход in_progress → done должен быть разрешён: %v", err)
		}
	})

	t.Run("SetTaskStatus_InvalidTransition", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{UserID: 70, Title: "A", Status: TaskDone})

		err := s.SetTaskStatus(ctx, 70, id, TaskInProgress)
		if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("ожидали ErrInvalidTransition, получили %v", err)
		}
	})

	t.Run("SetTaskStatus_NotFound", func(t *testing.T) {
		err := s.SetTaskStatus(ctx, 70, 999, TaskDone)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("ожидали ErrTaskNotFound, получили %v", err)
		}
	})

	// --- Изоляция копий ---

	t.Run("GetUser_ReturnsCopy", func(t *testing.T) {
		_ = s.SaveUser(ctx, &User{ID: 80, Consent: true})

		u, _ := s.GetUser(ctx, 80)
		u.Consent = false

		u2, _ := s.GetUser(ctx, 80)
		if !u2.Consent {
			t.Fatalf("изменение копии повлияло на хранилище")
		}
	})

	// --- Напоминания ---

	t.Run("GetTasksDueBefore_OnlyCloseDeadline", func(t *testing.T) {
		now := time.Now()

		_, _ = s.CreateTask(ctx, &Task{
			UserID:   90,
			Title:    "скоро",
			Deadline: now.Add(1 * time.Hour),
			Status:   TaskNew,
		})
		_, _ = s.CreateTask(ctx, &Task{
			UserID:   90,
			Title:    "не скоро",
			Deadline: now.Add(100 * time.Hour),
			Status:   TaskNew,
		})

		tasks, err := s.GetTasksDueBefore(ctx, now.Add(24*time.Hour))
		if err != nil {
			t.Fatalf("GetTasksDueBefore вернул ошибку: %v", err)
		}

		var found bool
		for _, task := range tasks {
			if task.UserID == 90 && task.Title == "скоро" {
				found = true
			}
			if task.UserID == 90 && task.Title == "не скоро" {
				t.Fatalf("вернулась задача с далёким дедлайном")
			}
		}
		if !found {
			t.Fatalf("не вернулась задача с близким дедлайном")
		}
	})

	t.Run("GetTasksDueBefore_SkipsDone", func(t *testing.T) {
		now := time.Now()

		_, _ = s.CreateTask(ctx, &Task{
			UserID:   91,
			Title:    "выполнена",
			Deadline: now.Add(1 * time.Hour),
			Status:   TaskDone,
		})

		tasks, err := s.GetTasksDueBefore(ctx, now.Add(24*time.Hour))
		if err != nil {
			t.Fatalf("GetTasksDueBefore вернул ошибку: %v", err)
		}
		for _, task := range tasks {
			if task.UserID == 91 && task.Title == "выполнена" {
				t.Fatalf("вернулась выполненная задача")
			}
		}
	})

	t.Run("GetTasksDueBefore_SkipsReminded", func(t *testing.T) {
		now := time.Now()

		id, _ := s.CreateTask(ctx, &Task{
			UserID:   92,
			Title:    "уже напомнили",
			Deadline: now.Add(1 * time.Hour),
			Status:   TaskNew,
		})
		if err := s.SetTaskReminderSent(ctx, id); err != nil {
			t.Fatalf("SetTaskReminderSent вернул ошибку: %v", err)
		}

		tasks, err := s.GetTasksDueBefore(ctx, now.Add(24*time.Hour))
		if err != nil {
			t.Fatalf("GetTasksDueBefore вернул ошибку: %v", err)
		}
		for _, task := range tasks {
			if task.UserID == 92 {
				t.Fatalf("вернулась задача, по которой уже напомнили")
			}
		}
	})

	t.Run("GetTasksDueBefore_SkipsZeroDeadline", func(t *testing.T) {
		now := time.Now()

		_, _ = s.CreateTask(ctx, &Task{
			UserID: 93,
			Title:  "без дедлайна",
			Status: TaskNew,
		})

		tasks, err := s.GetTasksDueBefore(ctx, now.Add(24*time.Hour))
		if err != nil {
			t.Fatalf("GetTasksDueBefore вернул ошибку: %v", err)
		}
		for _, task := range tasks {
			if task.UserID == 93 {
				t.Fatalf("вернулась задача без дедлайна")
			}
		}
	})

	t.Run("SetTaskReminderSent", func(t *testing.T) {
		id, _ := s.CreateTask(ctx, &Task{
			UserID:   94,
			Title:    "напомнить",
			Deadline: time.Now().Add(1 * time.Hour),
			Status:   TaskNew,
		})

		if err := s.SetTaskReminderSent(ctx, id); err != nil {
			t.Fatalf("SetTaskReminderSent вернул ошибку: %v", err)
		}

		got, _ := s.GetTask(ctx, 94, id)
		if !got.ReminderSent {
			t.Fatalf("ReminderSent не стал true")
		}
	})
}
