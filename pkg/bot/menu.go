package bot

import (
	"strconv"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// cbBtn — короткий хелпер: callback-кнопка.
func cbBtn(text, payload string) schemes.CallbackButton {
	return schemes.CallbackButton{
		Button: schemes.Button{
			Type: schemes.ButtonType("callback"),
			Text: text,
		},
		Payload: payload,
	}
}

func kbFromRows(rows ...[]schemes.ButtonInterface) *maxbot.Keyboard {
	return maxbot.InlineKeyboard(rows...)
}

func kbConsent() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("Да ✅", CbConsentYes),
			cbBtn("Нет ❌", CbConsentNo),
		},
	)
}

func kbReminders() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("Да", CbRemindYes),
			cbBtn("Нет", CbRemindNo),
		},
	)
}

func kbMainMenu() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("➕ Добавить задачу", CbMenuAdd),
			cbBtn("📋 Мои задачи", CbMenuList),
		},
		[]schemes.ButtonInterface{
			cbBtn("✏️ Изменить", CbMenuEdit),
			cbBtn("🗑 Удалить", CbMenuDelete),
		},
		[]schemes.ButtonInterface{
			cbBtn("✅ Выполнено", CbMenuDone),
		},
	)
}

func kbMood() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("😊", CbMoodGood),
			cbBtn("😐", CbMoodOK),
			cbBtn("😫", CbMoodBad),
		},
	)
}

// kbTaskSelect — список задач для выбора.
func kbTaskSelect(tasks []storage.Task, action string) *maxbot.Keyboard {
	rows := make([][]schemes.ButtonInterface, 0, len(tasks)+1)
	for i, t := range tasks {
		title := t.Title
		if runes := []rune(title); len(runes) > 30 {
			title = string(runes[:30]) + "…"
		}
		text := strconv.Itoa(i+1) + ". " + title
		payload := "task:" + action + ":" + strconv.FormatInt(t.ID, 10)
		rows = append(rows, []schemes.ButtonInterface{cbBtn(text, payload)})
	}
	rows = append(rows, []schemes.ButtonInterface{
		cbBtn("⬅️ Отмена", CbMenuList),
	})
	return kbFromRows(rows...)
}

func kbDeleteConfirm(taskID int64) *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("🗑 Да, удалить", cbTaskDelConfirm(taskID)),
			cbBtn("⬅️ Отмена", CbMenuList),
		},
	)
}

func kbPriority() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("🔥 Важно и срочно", "priority:1"),
			cbBtn("⭐ Важно", "priority:2"),
		},
		[]schemes.ButtonInterface{
			cbBtn("⏰ Срочно", "priority:3"),
			cbBtn("🌿 Обычное", "priority:4"),
		},
	)
}

func kbEditField() *maxbot.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("📝 Название", "edit:title"),
			cbBtn("📅 Дедлайн", "edit:deadline"),
		},
		[]schemes.ButtonInterface{
			cbBtn("🎯 Приоритет", "edit:priority"),
			cbBtn("⬅️ Отмена", CbMenuList),
		},
	)
}
