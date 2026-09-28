package bot

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// Формат даты, который ждём от пользователя.
const deadlineLayout = "02.01.2006"

// onTaskTitle — шаг 1: получили название задачи.
func (b *Bot) onTaskTitle(ctx context.Context, upd *schemes.MessageCreatedUpdate, text string) {
	userID := upd.Message.Sender.UserId

	if len([]rune(text)) > 200 {
		b.reply(ctx, upd, "Слишком длинное название (макс. 200). Попробуй короче.", nil)
		return
	}

	d := b.fsm.Draft(userID)
	d.Title = text
	b.fsm.Set(userID, StateAwaitTaskDeadline)

	b.reply(ctx, upd,
		"Когда дедлайн? Напиши дату в формате ДД.ММ.ГГГГ\nНапример: 15.06.2026",
		nil)
}

// onTaskDeadline — шаг 2: получили дату.
func (b *Bot) onTaskDeadline(ctx context.Context, upd *schemes.MessageCreatedUpdate, text string) {
	userID := upd.Message.Sender.UserId

	deadline, err := time.Parse(deadlineLayout, strings.TrimSpace(text))
	if err != nil {
		b.reply(ctx, upd,
			"Не понял дату. Напиши в формате ДД.ММ.ГГГГ, например 15.06.2026",
			nil)
		return
	}

	// Прошлое — не принимаем.
	if deadline.Before(time.Now().Add(-24 * time.Hour)) {
		b.reply(ctx, upd, "Дата уже прошла. Напиши будущую, пожалуйста.", nil)
		return
	}

	d := b.fsm.Draft(userID)
	d.Deadline = deadline.Format(deadlineLayout)
	b.fsm.Set(userID, StateAwaitTaskPriority)

	b.reply(ctx, upd, "Насколько задача важна и срочна?", kbPriority())
}

// onPriority — шаг 3: получили приоритет (callback из kbPriority).
// id ∈ {1, 2, 3, 4}:
//
//	1 — важно и срочно
//	2 — важно
//	3 — срочно
//	4 — обычное
func (b *Bot) onPriority(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	// Пользователь мог нажать кнопку не в том состоянии — вежливо выходим.
	if b.fsm.Get(userID) != StateAwaitTaskPriority {
		b.sendToUser(ctx, userID, "Сценарий добавления уже неактивен. Вот меню:", kbMainMenu())
		return
	}

	d := b.fsm.Draft(userID)
	d.Priority = int(id)

	// Разбираем priority → Urgent + Important.
	var urgent, important bool
	switch id {
	case 1:
		urgent, important = true, true
	case 2:
		urgent, important = false, true
	case 3:
		urgent, important = true, false
	case 4:
		urgent, important = false, false
	default:
		b.sendToUser(ctx, userID, "Не понял приоритет. Выбери кнопкой:", kbPriority())
		return
	}

	deadline, err := time.Parse(deadlineLayout, d.Deadline)
	if err != nil {
		// Теоретически невозможно — Deadline уже провалидирован, но подстрахуемся.
		log.Printf("onPriority: не удалось разобрать Deadline %q: %v", d.Deadline, err)
		b.fsm.Reset(userID)
		b.sendToUser(ctx, userID, "Что-то сломалось. Начни заново: /start", kbMainMenu())
		return
	}

	task := &storage.Task{
		UserID:    userID,
		Title:     d.Title,
		Deadline:  deadline,
		Urgent:    urgent,
		Important: important,
		Status:    storage.TaskNew,
	}

	taskID, err := b.store.CreateTask(ctx, task)
	if err != nil {
		log.Printf("CreateTask: %v", err)
		b.fsm.Reset(userID)
		b.sendToUser(ctx, userID, "Не получилось сохранить задачу. Попробуй позже.", kbMainMenu())
		return
	}

	b.fsm.Reset(userID)

	b.sendToUser(ctx, userID,
		"✅ Задача добавлена:\n\n"+
			"📌 "+task.Title+"\n"+
			"📅 до "+task.Deadline.Format(deadlineLayout)+"\n"+
			"🎯 "+priorityLabel(task),
		kbMainMenu())

	log.Printf("Создана задача #%d для пользователя %d", taskID, userID)
}

// priorityLabel возвращает читаемую метку приоритета по Эйзенхауэру.
func priorityLabel(t *storage.Task) string {
	switch {
	case t.Important && t.Urgent:
		return "🔥 Важно и срочно"
	case t.Important:
		return "⭐ Важно"
	case t.Urgent:
		return "⏰ Срочно"
	default:
		return "🌿 Обычное"
	}
}
