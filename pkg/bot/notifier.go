package bot

import (
	"context"
	"log"

	"max_bot_api/pkg/storage"
)

// Notifier — заглушка уведомлений о новых задачах. Реальная отправка
// через api.Messages.Send — задача M5. Живёт в pkg/bot, чтобы httpapi
// не тянул bot внутрь себя циклически.
type Notifier struct{}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) NotifyTaskCreated(_ context.Context, userID int64, task *storage.Task) {
	log.Printf("notify: user=%d task=%d %q", userID, task.ID, task.Title)
}
