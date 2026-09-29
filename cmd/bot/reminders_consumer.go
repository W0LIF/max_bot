package main

import (
	"context"
	"log"

	maxbot "github.com/max-messenger/max-bot-api-client-go"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/storage"
)

// runRemindersConsumer читает задачи из канала, которые шедулер
// признал «пора напомнить», и отправляет их пользователю в MAX.
//
// Логика:
//   - выключенные напоминания (reminders_on == false) — пропускаем;
//   - текст сообщения — формат deadlineHint (pkg/bot/dialog_list.go);
//   - к сообщению прикладываем клавиатуру с одной задачей, чтобы
//     закрыть её одним тапом (callback task:done:<id>).
//
// Шедулер ставит ReminderSent=true ДО отправки — если процесс упадёт
// между SetTaskReminderSent и Send, напоминание потеряется. Так
// гарантируется «ровно одно напоминание», даже если MAX недоступен.
func runRemindersConsumer(
	ctx context.Context,
	store storage.Store,
	api *maxbot.Api,
	ch <-chan storage.Task,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			sendReminder(ctx, store, api, task)
		}
	}
}

func sendReminder(ctx context.Context, store storage.Store, api *maxbot.Api, task storage.Task) {
	user, err := store.GetUser(ctx, task.UserID)
	if err != nil {
		log.Printf("reminder: GetUser(%d): %v", task.UserID, err)
		return
	}
	if !user.RemindersOn {
		return
	}

	if err := bot.SendTaskReminder(ctx, api, task); err != nil {
		log.Printf("reminder: send to %d: %v", task.UserID, err)
	}
}
