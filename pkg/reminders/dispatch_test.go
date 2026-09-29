package reminders

import (
	"context"
	"testing"
	"time"

	"max_bot_api/pkg/storage"
)

func TestDispatchDueRespectsSettingsAndDoesNotRepeat(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	if err := store.SaveUser(ctx, &storage.User{ID: 1, RemindersOn: false}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveUser(ctx, &storage.User{ID: 2, RemindersOn: true}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Hour)
	for _, userID := range []int64{1, 2} {
		if _, err := store.CreateTask(ctx, &storage.Task{
			UserID:   userID,
			Title:    "Тестовая задача",
			Deadline: deadline,
			Status:   storage.TaskNew,
		}); err != nil {
			t.Fatal(err)
		}
	}

	var delivered []int64
	sender := func(_ context.Context, task storage.Task) error {
		delivered = append(delivered, task.UserID)
		return nil
	}
	sent, err := DispatchDue(ctx, store, deadline.Add(time.Hour), sender)
	if err != nil {
		t.Fatalf("DispatchDue: %v", err)
	}
	if sent != 1 || len(delivered) != 1 || delivered[0] != 2 {
		t.Fatalf("expected only enabled user 2 to be notified; sent=%d users=%v", sent, delivered)
	}

	sent, err = DispatchDue(ctx, store, deadline.Add(time.Hour), sender)
	if err != nil {
		t.Fatalf("second DispatchDue: %v", err)
	}
	if sent != 0 || len(delivered) != 1 {
		t.Fatalf("reminder was sent more than once; sent=%d users=%v", sent, delivered)
	}

	remaining, err := store.GetTasksDueBefore(ctx, deadline.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].UserID != 1 {
		t.Fatalf("disabled user's reminder should remain pending: %+v", remaining)
	}
}
