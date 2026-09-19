// api/webhook.go — POST /api/webhook
//
// Требования сборщика Vercel (@vercel/go) к файлу внутри api/:
//  1. НЕ `package main`, если в проекте есть go.mod — иначе билд падает с
//     "Please change `package main` to `package handler`".
//  2. Экспортируемая функция с сигнатурой (http.ResponseWriter, *http.Request) —
//     сборщик сам найдёт её через AST и обернёт в http.HandlerFunc.
//
// Имена функций в пределах api/ должны различаться: все файлы в этой папке
// входят в один Go-пакет, иначе будет "redeclared in this block".
//
// Логика — в pkg/bot.
package handler

import (
	"net/http"

	"max_bot_api/pkg/bot"
)

// Webhook — вебхук MAX.
func Webhook(w http.ResponseWriter, r *http.Request) {
	bot.HandleWebhook(w, r)
}
