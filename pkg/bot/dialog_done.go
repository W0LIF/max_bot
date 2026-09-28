package bot

import (
	"context"
	"log"

	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// showDoneList — пользователь нажал «✅ Выполнено».
// Показываем активные задачи (не done) кнопками task:done:<id>.
func (b *Bot) showDoneList(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	tasks, err := b.store.GetTasks(ctx, userID)
	if err != nil {
		log.Printf("GetTasks: %v", err)
		b.sendToUser(ctx, userID, "Не удалось получить список задач.", kbMainMenu())
		return
	}

	// Оставляем только те, что ещё не выполнены.
	active := make([]storage.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status != storage.TaskDone {
			active = append(active, t)
		}
	}

	if len(active) == 0 {
		b.sendToUser(ctx, userID,
			"🎉 Все задачи выполнены! Добавь новую — «➕ Добавить задачу».",
			kbMainMenu())
		return
	}

	b.sendToUser(ctx, userID,
		"Какую задачу отметить выполненной?",
		kbTaskSelect(active, "done"))
}

// onTaskDone — callback task:done:<id>. Меняем статус и спрашиваем настроение.
func (b *Bot) onTaskDone(ctx context.Context, upd *schemes.MessageCallbackUpdate, id int64) {
	userID := upd.Callback.User.UserId

	task, err := b.store.GetTask(ctx, userID, id)
	if err != nil {
		log.Printf("GetTask(%d): %v", id, err)
		b.sendToUser(ctx, userID, "Задача не найдена — возможно, её удалили.", kbMainMenu())
		return
	}

	if task.Status == storage.TaskDone {
		b.sendToUser(ctx, userID, "Эта задача уже выполнена.", kbMainMenu())
		return
	}

	if err := b.store.SetTaskStatus(ctx, userID, id, storage.TaskDone); err != nil {
		log.Printf("SetTaskStatus(%d): %v", id, err)
		b.sendToUser(ctx, userID, "Не получилось обновить статус. Попробуй позже.", kbMainMenu())
		return
	}

	// Спрашиваем настроение — шаг 7 сценария из концепции.
	b.sendToUser(ctx, userID,
		"✅ «"+task.Title+"» — выполнено!\n\nКак ты себя чувствуешь после этого?",
		kbMood())
}

// onMood — callback mood:good / mood:ok / mood:bad.
// Пока просто подтверждаем и возвращаем в меню. Поле для хранения
// настроения появится позже — это общая задача с Разр1.
func (b *Bot) onMood(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId

	reply := "Записал. Спасибо!"
	switch upd.Callback.Payload {
	case CbMoodGood:
		reply = "😊 Отлично! Так держать."
	case CbMoodOK:
		reply = "😐 Принял. Не забывай про отдых."
	case CbMoodBad:
		reply = "😫 Понял. Если тяжело — сделай паузу, ты не обязан(а) всё сразу."
	}

	b.sendToUser(ctx, userID, reply, kbMainMenu())
}
