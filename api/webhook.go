package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Инициализируем клиента MAX (токен из переменной окружения)
	api, err := maxbot.New(os.Getenv("MAX_BOT_TOKEN"))
	if err != nil {
		log.Printf("Ошибка инициализации бота: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var update schemes.UpdateInterface
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("Ошибка декодирования: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Обрабатываем только новые сообщения
	if msgUpdate, ok := update.(*schemes.MessageCreatedUpdate); ok {
		text := msgUpdate.Message.Body.Text
		chatID := msgUpdate.Message.Recipient.ChatId

		ctx := r.Context()

		if text == "/start" {
			msg := maxbot.NewMessage().
				SetChat(chatID).
				SetText("Привет! Я бот для учёбы. Открой мини-приложение, чтобы начать.")
			if err := api.Messages.Send(ctx, msg); err != nil {
				log.Printf("Ошибка отправки: %v", err)
			}
		} else {
			msg := maxbot.NewMessage().
				SetChat(chatID).
				SetText(fmt.Sprintf("Вы написали: %s", text))
			if err := api.Messages.Send(ctx, msg); err != nil {
				log.Printf("Ошибка отправки: %v", err)
			}
		}
	}

	// MAX требует, чтобы вебхук возвращал 200 OK
	w.WriteHeader(http.StatusOK)
}
