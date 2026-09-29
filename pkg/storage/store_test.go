package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- runStoreTests: базовая проверка user + task ---

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
		if err := s.SetTaskReminderSent(ctx, id); !errors.Is(err, ErrReminderAlreadySent) {
			t.Fatalf("повторная отметка должна отклоняться, получили %v", err)
		}
		if err := s.ResetTaskReminderSent(ctx, id); err != nil {
			t.Fatalf("ResetTaskReminderSent вернул ошибку: %v", err)
		}
		if err := s.SetTaskReminderSent(ctx, id); err != nil {
			t.Fatalf("повторная попытка после сброса не удалась: %v", err)
		}
	})
}

// --- runExtendedStoreTests: moods/notes/feedback ---

func runExtendedStoreTests(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()

	t.Run("CreateAndGetMoods", func(t *testing.T) {
		m := &Mood{UserID: 100, Value: MoodGood, Note: "ок"}
		id, err := s.CreateMood(ctx, m)
		if err != nil {
			t.Fatalf("CreateMood: %v", err)
		}
		if id == 0 {
			t.Fatal("CreateMood вернул 0")
		}

		moods, err := s.GetMoods(ctx, 100, 7)
		if err != nil {
			t.Fatalf("GetMoods: %v", err)
		}
		if len(moods) != 1 {
			t.Fatalf("ожидали 1 запись, получили %d", len(moods))
		}
		if moods[0].Value != MoodGood {
			t.Fatalf("Value не совпадает: %q", moods[0].Value)
		}
	})

	t.Run("GetMoods_FiltersOldEntries", func(t *testing.T) {
		old := &Mood{
			UserID:    101,
			Value:     MoodBad,
			CreatedAt: time.Now().AddDate(0, 0, -30),
		}
		if _, err := s.CreateMood(ctx, old); err != nil {
			t.Fatalf("CreateMood: %v", err)
		}

		moods, err := s.GetMoods(ctx, 101, 7)
		if err != nil {
			t.Fatalf("GetMoods: %v", err)
		}
		if len(moods) != 0 {
			t.Fatalf("старая запись не должна попасть в окно 7 дней, получили %d", len(moods))
		}
	})

	t.Run("GetMoods_IsolatesUsers", func(t *testing.T) {
		_, _ = s.CreateMood(ctx, &Mood{UserID: 102, Value: MoodGood})
		_, _ = s.CreateMood(ctx, &Mood{UserID: 103, Value: MoodBad})

		moods, _ := s.GetMoods(ctx, 102, 7)
		for _, m := range moods {
			if m.UserID != 102 {
				t.Fatalf("вернулась чужая запись: %+v", m)
			}
		}
	})

	t.Run("CreateAndGetNotes", func(t *testing.T) {
		start := time.Now().Add(1 * time.Hour)
		end := start.Add(1 * time.Hour)
		id, err := s.CreateNote(ctx, &Note{
			UserID: 110,
			Text:   "встреча",
			Start:  start,
			End:    end,
		})
		if err != nil {
			t.Fatalf("CreateNote: %v", err)
		}
		if id == 0 {
			t.Fatal("CreateNote вернул 0")
		}

		notes, err := s.GetNotes(ctx, 110)
		if err != nil {
			t.Fatalf("GetNotes: %v", err)
		}
		if len(notes) != 1 || notes[0].Text != "встреча" {
			t.Fatalf("данные заметки не совпадают: %+v", notes)
		}
	})

	t.Run("DeleteNote", func(t *testing.T) {
		id, _ := s.CreateNote(ctx, &Note{
			UserID: 111,
			Text:   "удалить",
			Start:  time.Now(),
			End:    time.Now().Add(time.Hour),
		})
		if err := s.DeleteNote(ctx, 111, id); err != nil {
			t.Fatalf("DeleteNote: %v", err)
		}
		notes, _ := s.GetNotes(ctx, 111)
		if len(notes) != 0 {
			t.Fatalf("заметка не удалилась")
		}
	})

	t.Run("DeleteNote_NotFound", func(t *testing.T) {
		err := s.DeleteNote(ctx, 111, 99999)
		if !errors.Is(err, ErrNoteNotFound) {
			t.Fatalf("ожидали ErrNoteNotFound, получили %v", err)
		}
	})

	t.Run("DeleteNote_WrongUser", func(t *testing.T) {
		id, _ := s.CreateNote(ctx, &Note{
			UserID: 112,
			Text:   "не трогать",
			Start:  time.Now(),
			End:    time.Now().Add(time.Hour),
		})
		err := s.DeleteNote(ctx, 113, id)
		if !errors.Is(err, ErrNoteNotFound) {
			t.Fatalf("ожидали ErrNoteNotFound, получили %v", err)
		}
	})

	t.Run("CreateFeedback", func(t *testing.T) {
		id, err := s.CreateFeedback(ctx, &Feedback{UserID: 120, Text: "всё ок"})
		if err != nil {
			t.Fatalf("CreateFeedback: %v", err)
		}
		if id == 0 {
			t.Fatal("CreateFeedback вернул 0")
		}
	})
}

// --- runGroupStoreTests: группы ---

func runGroupStoreTests(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()

	t.Run("GetUserGroup_NotFound", func(t *testing.T) {
		_, err := s.GetUserGroup(ctx, 999)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("ожидали ErrGroupNotFound, получили %v", err)
		}
	})

	t.Run("EnsureAndGetUserGroup", func(t *testing.T) {
		if err := s.EnsureGroup(ctx, 200, DemoGroupID); err != nil {
			t.Fatalf("EnsureGroup: %v", err)
		}
		gid, err := s.GetUserGroup(ctx, 200)
		if err != nil {
			t.Fatalf("GetUserGroup: %v", err)
		}
		if gid != DemoGroupID {
			t.Fatalf("ожидали groupID=%d, получили %d", DemoGroupID, gid)
		}
	})

	t.Run("EnsureGroup_Idempotent", func(t *testing.T) {
		_ = s.EnsureGroup(ctx, 201, DemoGroupID)
		if err := s.EnsureGroup(ctx, 201, DemoGroupID); err != nil {
			t.Fatalf("повторный EnsureGroup: %v", err)
		}
		gid, _ := s.GetUserGroup(ctx, 201)
		if gid != DemoGroupID {
			t.Fatalf("группа изменилась: %d", gid)
		}
	})

	t.Run("LeaveGroups", func(t *testing.T) {
		if err := s.EnsureGroup(ctx, 202, DemoGroupID); err != nil {
			t.Fatalf("EnsureGroup: %v", err)
		}
		if err := s.LeaveGroups(ctx, 202); err != nil {
			t.Fatalf("LeaveGroups: %v", err)
		}
		if _, err := s.GetUserGroup(ctx, 202); !errors.Is(err, ErrGroupNotFound) {
			t.Fatalf("ожидали выход из группы, получили %v", err)
		}
	})

	t.Run("GetGroupMembers", func(t *testing.T) {
		_ = s.SaveUser(ctx, &User{ID: 210, Name: "Аня"})
		_ = s.SaveUser(ctx, &User{ID: 211, Name: "Борис"})
		_ = s.EnsureGroup(ctx, 210, DemoGroupID)
		_ = s.EnsureGroup(ctx, 211, DemoGroupID)

		_, _ = s.CreateTask(ctx, &Task{
			UserID: 210,
			Title:  "активная",
			Status: TaskNew,
		})

		members, err := s.GetGroupMembers(ctx, DemoGroupID)
		if err != nil {
			t.Fatalf("GetGroupMembers: %v", err)
		}

		var found bool
		for _, m := range members {
			if m.ID == 210 {
				found = true
				if m.Tasks < 1 {
					t.Fatalf("у Ани должна быть активная задача, получили %d", m.Tasks)
				}
			}
		}
		if !found {
			t.Fatal("Аня не найдена в членах группы")
		}
	})
}
