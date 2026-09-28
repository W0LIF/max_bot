package bot

import "strconv"

// Формат callback-данных (согласовано с Разр1):
//
//	consent:yes | consent:no
//	remind:yes  | remind:no
//	menu:add | menu:list | menu:edit | menu:delete | menu:done
//	task:edit:<id> | task:delete:<id> | task:done:<id> | task:delconfirm:<id>
//	mood:good | mood:ok | mood:bad
//	priority:1 | priority:2 | priority:3 | priority:4
const (
	CbConsentYes = "consent:yes"
	CbConsentNo  = "consent:no"

	CbRemindYes = "remind:yes"
	CbRemindNo  = "remind:no"

	CbMenuAdd    = "menu:add"
	CbMenuList   = "menu:list"
	CbMenuEdit   = "menu:edit"
	CbMenuDelete = "menu:delete"
	CbMenuDone   = "menu:done"

	CbMoodGood = "mood:good" // 😊
	CbMoodOK   = "mood:ok"   // 😐
	CbMoodBad  = "mood:bad"  // 😫
)

// parseCallback разбирает строку вида "task:done:42".
//
//	"menu:add"       -> ("menu:add", 0, true)
//	"task:done:42"   -> ("task:done", 42, true)
//	"priority:2"     -> ("priority", 2, true)
//	"мусор"          -> ("", 0, false)
func parseCallback(data string) (action string, id int64, ok bool) {
	if data == "" {
		return "", 0, false
	}

	// Ищем последний ":" — он отделяет числовой ID, если он есть.
	last := -1
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == ':' {
			last = i
			break
		}
	}

	// Нет ":" вообще — просто action без id.
	if last == -1 {
		return data, 0, true
	}

	// Есть ":" — проверяем, число ли после него.
	rawID := data[last+1:]
	n, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		// "consent:yes" — ":" есть, но после него не число.
		// Значит action = вся строка, id = 0.
		return data, 0, true
	}

	return data[:last], n, true
}

// Хелперы для сборки callback-строк с ID задачи.

func cbTaskDone(id int64) string       { return "task:done:" + strconv.FormatInt(id, 10) }
func cbTaskEdit(id int64) string       { return "task:edit:" + strconv.FormatInt(id, 10) }
func cbTaskDelete(id int64) string     { return "task:delete:" + strconv.FormatInt(id, 10) }
func cbTaskDelConfirm(id int64) string { return "task:delconfirm:" + strconv.FormatInt(id, 10) }
