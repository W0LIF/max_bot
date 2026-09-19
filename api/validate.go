// api/validate.go — POST /api/validate
//
// Требования сборщика Vercel (@vercel/go) такие же, как для webhook.go:
// пакет не `main` + экспортируемая функция (http.ResponseWriter, *http.Request).
//
// Имя функции отличается от имени в webhook.go, потому что оба файла
// лежат в одном Go-пакете.
//
// Логика — в pkg/bot.
package handler

import (
	"net/http"

	"max_bot_api/pkg/bot"
)

// Validate — проверка подписи initData мини-приложения MAX.
func Validate(w http.ResponseWriter, r *http.Request) {
	bot.HandleValidate(w, r)
}
