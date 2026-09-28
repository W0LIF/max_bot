package handler

import (
	"net/http"

	"max_bot_api/pkg/bot"
)

func Validate(w http.ResponseWriter, r *http.Request) {
	bot.HandleValidate(w, r)
}
