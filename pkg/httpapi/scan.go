package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var knownSubjects = []string{
	"Математика", "Физика", "Информатика", "История", "Химия", "Биология",
	"Русский язык", "Английский язык", "Программирование",
}

// handleScan — заглушка распознавания фото. Файл никуда не сохраняется
// и не отправляется. Ответ детерминирован: заголовок из имени файла,
// subject — по справочнику, deadline — ближайший рабочий день.
// Настоящий OCR — вне MVP, формат ответа не изменится.
func handleScan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать форму")
			return
		}
		file, header, err := r.FormFile("photo")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Поле photo обязательно")
			return
		}
		defer file.Close()

		// Файл не сохраняем — заглушка.
		title := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		title = strings.TrimSpace(title)
		if title == "" {
			title = "Новая задача"
		}

		subject := detectSubject(title)
		deadline := nextWeekday(time.Now())

		writeJSON(w, http.StatusOK, map[string]any{
			"title":    title,
			"subject":  subject,
			"deadline": deadline.UTC().Format(time.RFC3339),
		})
	}
}

func detectSubject(title string) string {
	low := strings.ToLower(title)
	for _, s := range knownSubjects {
		if strings.Contains(low, strings.ToLower(s)) {
			return s
		}
	}
	return "Общее"
}

// nextWeekday — следующий рабочий день (пн–пт) в 12:00 UTC.
// Если today — пятница, вернёт понедельник.
func nextWeekday(from time.Time) time.Time {
	t := from.AddDate(0, 0, 1)
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.AddDate(0, 0, 1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC)
}
