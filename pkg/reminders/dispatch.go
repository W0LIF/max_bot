package reminders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"max_bot_api/pkg/storage"
)

type ReminderSender func(context.Context, storage.Task) error

// DispatchDue sends eligible reminders once, leaving disabled users' tasks
// pending so enabling reminders later still allows delivery.
func DispatchDue(ctx context.Context, store storage.Store, before time.Time, send ReminderSender) (int, error) {
	if send == nil {
		return 0, errors.New("reminder sender is required")
	}

	tasks, err := store.GetTasksDueBefore(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("get due tasks: %w", err)
	}

	sent := 0
	var failures []error
	for _, task := range tasks {
		user, err := store.GetUser(ctx, task.UserID)
		if errors.Is(err, storage.ErrUserNotFound) {
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("get user %d: %w", task.UserID, err))
			continue
		}
		if !user.RemindersOn {
			continue
		}
		if err := store.SetTaskReminderSent(ctx, task.ID); err != nil {
			if errors.Is(err, storage.ErrReminderAlreadySent) {
				continue
			}
			failures = append(failures, fmt.Errorf("mark task %d reminded: %w", task.ID, err))
			continue
		}
		if err := send(ctx, task); err != nil {
			failures = append(failures, fmt.Errorf("send task %d reminder: %w", task.ID, err))
			continue
		}
		sent++
	}

	return sent, errors.Join(failures...)
}
