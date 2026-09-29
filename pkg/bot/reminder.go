package bot

import (
	"context"
	"fmt"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

func SendTaskReminder(ctx context.Context, api *maxbot.Api, task storage.Task) error {
	text := fmt.Sprintf(
		"⏰ %s дедлайн по задаче\n\n📌 %s\n📅 до %s",
		humanDeadlineHint(task.Deadline),
		task.Title,
		task.Deadline.Format("02.01.2006"),
	)
	message := maxbot.NewMessage().SetUser(task.UserID).SetText(text)
	message.AddKeyboard(reminderKeyboard(task))
	return api.Messages.Send(ctx, message)
}

func humanDeadlineHint(deadline time.Time) string {
	days := int(time.Until(deadline).Hours() / 24)
	switch {
	case days < 0:
		return "Просрочен"
	case days == 0:
		return "Сегодня"
	case days == 1:
		return "Завтра"
	default:
		return fmt.Sprintf("Через %d дн.", days)
	}
}

func reminderKeyboard(task storage.Task) *maxbot.Keyboard {
	return maxbot.InlineKeyboard(
		[]schemes.ButtonInterface{
			schemes.CallbackButton{
				Button:  schemes.Button{Type: schemes.CALLBACK, Text: "✅ Выполнено"},
				Payload: cbTaskDone(task.ID),
				Intent:  schemes.DEFAULT,
			},
		},
	)
}
