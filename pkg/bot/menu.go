package bot

import (
	"strconv"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

// Все клавиатуры бота собраны здесь.
//
// API библиотеки (проверено через go doc):
//   - schemes.Keyboard{ Buttons [][]ButtonInterface }
//   - schemes.CallbackButton{ Button{Type, Text}, Payload string }
//   - тип кнопки — schemes.ButtonType (строка)

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

// kbFromRows собирает Keyboard из рядов кнопок.
// Каждый ряд — срез ButtonInterface, поэтому CallbackButton
// автоматически приводится к интерфейсу.
func kbFromRows(rows ...[]schemes.ButtonInterface) *schemes.Keyboard {
	return &schemes.Keyboard{Buttons: rows}
}

// kbConsent — согласие на обработку данных.
func kbConsent() *schemes.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("Да ✅", CbConsentYes),
			cbBtn("Нет ❌", CbConsentNo),
		},
	)
}

// kbReminders — вопрос о напоминаниях.
func kbReminders() *schemes.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("Да", CbRemindYes),
			cbBtn("Нет", CbRemindNo),
		},
	)
}

// kbMainMenu — главное меню.
func kbMainMenu() *schemes.Keyboard {
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

// kbMood — трекинг настроения после выполнения задачи.
func kbMood() *schemes.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("😊", CbMoodGood),
			cbBtn("😐", CbMoodOK),
			cbBtn("😫", CbMoodBad),
		},
	)
}

// kbTaskSelect — список задач для выбора.
// action: "done" | "edit" | "delete" → payload "task:<action>:<id>".
func kbTaskSelect(tasks []Task, action string) *schemes.Keyboard {
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

// kbDeleteConfirm — подтверждение удаления задачи.
func kbDeleteConfirm(taskID int64) *schemes.Keyboard {
	return kbFromRows(
		[]schemes.ButtonInterface{
			cbBtn("🗑 Да, удалить", cbTaskDelConfirm(taskID)),
			cbBtn("⬅️ Отмена", CbMenuList),
		},
	)
}

// kbPriority — приоритет по Эйзенхауэру.
func kbPriority() *schemes.Keyboard {
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
