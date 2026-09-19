package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные системы")
	}

	token := os.Getenv("MAX_BOT_TOKEN")
	ctx := context.Background()

	api, err := maxbot.New(token)
	if err != nil {
		log.Fatalf("Ошибка инициализации: %v", err)
	}

	botInfo, err := api.Bots.GetBot(ctx)
	if err != nil {
		log.Fatalf("Не удалось получить информацию о боте: %v", err)
	}
	fmt.Printf("Бот запущен: %s (ID: %d, Username: %s)\n", botInfo.Name, botInfo.UserId, botInfo.Username)

	// HTTP-сервер для мини-приложения
	http.HandleFunc("/api/validate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req struct {
			InitData string `json:"initData"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if req.InitData == "" {
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Сервер работает! Откройте приложение внутри MAX для проверки подписи.",
			})
			return
		}

		valid, err := ValidateInitData(req.InitData, token)
		if err != nil || !valid {
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Данные недействительны",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Данные подтверждены! Пользователь аутентифицирован.",
		})
	})

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("HTTP-сервер запущен на порту %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Ошибка HTTP-сервера: %v", err)
		}
	}()

	updates := api.GetUpdates(ctx)
	for update := range updates {
		switch upd := update.(type) {
		case *schemes.MessageCreatedUpdate:
			text := upd.Message.Body.Text
			chatID := upd.Message.Recipient.ChatId

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
	}
}
