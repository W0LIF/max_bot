package main

import (
	"context"
	"fmt"
	"log"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"

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

	text := formatReminderText(task)

	msg := maxbot.NewMessage().SetUser(task.UserID).SetText(text)
	msg.AddKeyboard(reminderKeyboard(task))

	if err := api.Messages.Send(ctx, msg); err != nil {
		log.Printf("reminder: send to %d: %v", task.UserID, err)
	}
}

// formatReminderText — «⏰ Через N дней дедлайн по задаче …».
// Формат близок к deadlineHint из pkg/bot/dialog_list.go.
func formatReminderText(task storage.Task) string {
	hint := humanDeadlineHint(task.Deadline)
	return fmt.Sprintf(
		"⏰ %s дедлайн по задаче\n\n📌 %s\n📅 до %s",
		hint,
		task.Title,
		task.Deadline.Format("02.01.2006"),
	)
}

// humanDeadlineHint — сколько осталось до дедлайна. Дублирует логику
// deadlineHint из pkg/bot, но та не экспортирована и живёт в другом
// пакете. Копия сознательная: тексты пушей и списка задач могут
// развиваться независимо.
func humanDeadlineHint(deadline time.Time) string {
	now := time.Now()
	days := int(deadline.Sub(now).Hours() / 24)

	switch {
	case days < 0:
		return "Просрочен"
	case days == 0:
		return "Сегодня"
	case days == 1:
		return "Завтра"
	case days <= 7:
		return fmt.Sprintf("Через %d дн.", days)
	default:
		return fmt.Sprintf("Через %d дн.", days)
	}
}

// reminderKeyboard — одна кнопка «✅ Выполнено» с callback
// task:done:<id>, чтобы из уведомления задачу можно было закрыть.
func reminderKeyboard(task storage.Task) *maxbot.Keyboard {
	payload := fmt.Sprintf("task:done:%d", task.ID)
	return maxbot.InlineKeyboard(
		[]schemes.ButtonInterface{
			schemes.CallbackButton{
				Button: schemes.Button{
					Type: schemes.CALLBACK,
					Text: "✅ Выполнено",
				},
				Payload: payload,
				Intent:  schemes.DEFAULT,
			},
		},
	)
}
