package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// showTaskList — сценарий «Мои задачи»: вывести список с их статусами
// и дедлайнами. Данные приходят уже отсортированными из Store.GetTasks
// (сначала важно+срочно, потом по дедлайну).
func (b *Bot) showTaskList(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	tasks, err := b.store.GetTasks(ctx, userID)
	if err != nil {
		log.Printf("GetTasks: %v", err)
		b.sendToUser(ctx, userID, "Не удалось получить список задач.", kbMainMenu())
		return
	}

	if len(tasks) == 0 {
		b.sendToUser(ctx, userID,
			"📋 Список пуст. Добавь первую задачу — кнопка «➕ Добавить задачу».",
			kbMainMenu())
		return
	}

	// Делим по статусу.
	var active, done []storage.Task
	for _, t := range tasks {
		if t.Status == storage.TaskDone {
			done = append(done, t)
		} else {
			active = append(active, t)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Твои задачи (всего %d):\n", len(tasks)))

	if len(active) > 0 {
		sb.WriteString("\n🟡 Активные:\n")
		for i, t := range active {
			sb.WriteString(formatTaskLine(i+1, t))
		}
	}

	if len(done) > 0 {
		sb.WriteString("\n✅ Выполненные:\n")
		for i, t := range done {
			sb.WriteString(formatTaskLine(i+1, t))
		}
	}

	b.sendToUser(ctx, userID, sb.String(), kbMainMenu())
}

// formatTaskLine — одна строка задачи для списка.
//
// Пример:
//
//  1. 🔥 Сдать лабу
//     📅 до 15.06.2026 · ⏰ осталось 3 дня
func formatTaskLine(num int, t storage.Task) string {
	var sb strings.Builder

	icon := "🌿"
	switch {
	case t.Important && t.Urgent:
		icon = "🔥"
	case t.Important:
		icon = "⭐"
	case t.Urgent:
		icon = "⏰"
	}

	statusIcon := ""
	switch t.Status {
	case storage.TaskInProgress:
		statusIcon = "🔵 "
	case storage.TaskDone:
		statusIcon = "✅ "
	}

	sb.WriteString(fmt.Sprintf("%d. %s%s %s\n", num, statusIcon, icon, t.Title))

	if !t.Deadline.IsZero() {
		sb.WriteString("   📅 до " + t.Deadline.Format(deadlineLayout))
		if t.Status != storage.TaskDone {
			sb.WriteString(" · " + deadlineHint(t.Deadline))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// deadlineHint — человеко-читаемое «сколько осталось».
func deadlineHint(deadline time.Time) string {
	now := time.Now()
	days := int(deadline.Sub(now).Hours() / 24)

	switch {
	case days < 0:
		return "⚠️ просрочено"
	case days == 0:
		return "🔥 сегодня"
	case days == 1:
		return "завтра"
	case days <= 7:
		return fmt.Sprintf("осталось %d дн.", days)
	default:
		return fmt.Sprintf("через %d дн.", days)
	}
}
