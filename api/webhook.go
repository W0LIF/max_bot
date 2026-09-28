package handler

import (
	"net/http"

	"max_bot_api/pkg/bot"
)

func Webhook(w http.ResponseWriter, r *http.Request) {
	bot.HandleWebhook(w, r)
}
