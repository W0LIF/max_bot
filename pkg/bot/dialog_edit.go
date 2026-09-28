package bot

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// --- Список задач для изменения ---

func (b *Bot) showEditList(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	tasks, err := b.store.GetTasks(ctx, userID)
	if err != nil {
		log.Printf("GetTasks: %v", err)
		b.sendToUser(ctx, userID, "Не удалось получить список задач.", kbMainMenu())
		return
	}

	active := make([]storage.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status != storage.TaskDone {
			active = append(active, t)
		}
	}

	if len(active) == 0 {
		b.sendToUser(ctx, userID, "Нет активных задач, которые можно изменить.", kbMainMenu())
		return
	}

	b.sendToUser(ctx, userID, "Какую задачу изменить?", kbTaskSelect(active, "edit"))
}

// onTaskEdit — callback task:edit:<id>. Запоминаем ID и спрашиваем, что менять.
func (b *Bot) onTaskEdit(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	task, err := b.store.GetTask(ctx, userID, id)
	if err != nil {
		log.Printf("GetTask(%d): %v", id, err)
		b.sendToUser(ctx, userID, "Задача не найдена.", kbMainMenu())
		return
	}

	d := b.fsm.Draft(userID)
	d.EditID = id

	b.sendToUser(ctx, userID,
		"✏️ «"+task.Title+"»\n\nЧто меняем?",
		kbEditField())
}

// --- Изменение названия ---

func (b *Bot) onEditFieldTitle(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	if b.fsm.Draft(userID).EditID == 0 {
		b.sendToUser(ctx, userID, "Сначала выбери задачу.", kbMainMenu())
		return
	}

	b.fsm.Set(userID, StateAwaitEditTitle)
	b.sendToUser(ctx, userID, "Напиши новое название:", nil)
}

func (b *Bot) onEditTitle(ctx context.Context, upd *schemes.MessageCreatedUpdate, text string) {
	userID := upd.Message.Sender.UserId

	if len([]rune(text)) > 200 {
		b.reply(ctx, upd, "Слишком длинное название (макс. 200).", nil)
		return
	}

	d := b.fsm.Draft(userID)
	task, err := b.store.GetTask(ctx, userID, d.EditID)
	if err != nil {
		b.fsm.Reset(userID)
		b.reply(ctx, upd, "Задача не найдена.", kbMainMenu())
		return
	}

	task.Title = text
	if err := b.store.UpdateTask(ctx, task); err != nil {
		log.Printf("UpdateTask: %v", err)
		b.fsm.Reset(userID)
		b.reply(ctx, upd, "Не удалось сохранить.", kbMainMenu())
		return
	}

	b.fsm.Reset(userID)
	b.reply(ctx, upd, "✅ Название обновлено: «"+text+"»", kbMainMenu())
}

// --- Изменение дедлайна ---

func (b *Bot) onEditFieldDeadline(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	if b.fsm.Draft(userID).EditID == 0 {
		b.sendToUser(ctx, userID, "Сначала выбери задачу.", kbMainMenu())
		return
	}

	b.fsm.Set(userID, StateAwaitEditDeadline)
	b.sendToUser(ctx, userID, "Напиши новый дедлайн в формате ДД.ММ.ГГГГ:", nil)
}

func (b *Bot) onEditDeadline(ctx context.Context, upd *schemes.MessageCreatedUpdate, text string) {
	userID := upd.Message.Sender.UserId

	deadline, err := time.Parse(deadlineLayout, strings.TrimSpace(text))
	if err != nil {
		b.reply(ctx, upd, "Не понял дату. Формат: ДД.ММ.ГГГГ", nil)
		return
	}

	d := b.fsm.Draft(userID)
	task, err := b.store.GetTask(ctx, userID, d.EditID)
	if err != nil {
		b.fsm.Reset(userID)
		b.reply(ctx, upd, "Задача не найдена.", kbMainMenu())
		return
	}

	task.Deadline = deadline
	if err := b.store.UpdateTask(ctx, task); err != nil {
		log.Printf("UpdateTask: %v", err)
		b.fsm.Reset(userID)
		b.reply(ctx, upd, "Не удалось сохранить.", kbMainMenu())
		return
	}

	b.fsm.Reset(userID)
	b.reply(ctx, upd, "✅ Дедлайн обновлён: "+deadline.Format(deadlineLayout), kbMainMenu())
}

// --- Изменение приоритета ---

func (b *Bot) onEditFieldPriority(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	if b.fsm.Draft(userID).EditID == 0 {
		b.sendToUser(ctx, userID, "Сначала выбери задачу.", kbMainMenu())
		return
	}

	b.fsm.Set(userID, StateAwaitEditPriority)
	b.sendToUser(ctx, userID, "Какой новый приоритет?", kbPriority())
}

func (b *Bot) onEditPriority(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	d := b.fsm.Draft(userID)
	task, err := b.store.GetTask(ctx, userID, d.EditID)
	if err != nil {
		b.fsm.Reset(userID)
		b.sendToUser(ctx, userID, "Задача не найдена.", kbMainMenu())
		return
	}

	switch id {
	case 1:
		task.Urgent, task.Important = true, true
	case 2:
		task.Urgent, task.Important = false, true
	case 3:
		task.Urgent, task.Important = true, false
	case 4:
		task.Urgent, task.Important = false, false
	default:
		b.sendToUser(ctx, userID, "Не понял приоритет.", kbPriority())
		return
	}

	if err := b.store.UpdateTask(ctx, task); err != nil {
		log.Printf("UpdateTask: %v", err)
		b.fsm.Reset(userID)
		b.sendToUser(ctx, userID, "Не удалось сохранить.", kbMainMenu())
		return
	}

	b.fsm.Reset(userID)
	b.sendToUser(ctx, userID, "✅ Приоритет обновлён: "+priorityLabel(task), kbMainMenu())
}

// --- Удаление ---

func (b *Bot) showDeleteList(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	tasks, err := b.store.GetTasks(ctx, userID)
	if err != nil {
		log.Printf("GetTasks: %v", err)
		b.sendToUser(ctx, userID, "Не удалось получить список задач.", kbMainMenu())
		return
	}

	if len(tasks) == 0 {
		b.sendToUser(ctx, userID, "Список пуст — нечего удалять.", kbMainMenu())
		return
	}

	b.sendToUser(ctx, userID, "Какую задачу удалить?", kbTaskSelect(tasks, "delete"))
}

func (b *Bot) onTaskDelete(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	task, err := b.store.GetTask(ctx, userID, id)
	if err != nil {
		b.sendToUser(ctx, userID, "Задача не найдена.", kbMainMenu())
		return
	}

	b.sendToUser(ctx, userID,
		"Удалить «"+task.Title+"»? Это действие нельзя отменить.",
		kbDeleteConfirm(id))
}

func (b *Bot) onTaskDeleteConfirm(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	if err := b.store.DeleteTask(ctx, userID, id); err != nil {
		log.Printf("DeleteTask(%d): %v", id, err)
		b.sendToUser(ctx, userID, "Не удалось удалить.", kbMainMenu())
		return
	}

	b.sendToUser(ctx, userID, "🗑 Задача удалена.", kbMainMenu())
}
