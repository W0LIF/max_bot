// Package bot содержит общую логику бота MAX: обработку апдейтов,
// ответы на сообщения и проверку подписи мини-приложения.
//
// Пакет намеренно лежит вне api/ и вне internal/:
//   - всё, что лежит в api/, Vercel считает отдельной serverless-функцией;
//   - internal-пакеты Vercel ломает, т.к. при сборке переписывает путь модуля.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const startAnswer = "Привет! Я бот для учёбы. Открой мини-приложение, чтобы начать."

// HandleWebhook — обработчик webhook от MAX (маршрут /api/webhook).
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		log.Println("MAX_BOT_TOKEN не задан")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	api, err := NewAPIClient(token)
	if err != nil {
		log.Printf("Ошибка инициализации бота: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Секрет из подписки (Subscriptions.Subscribe). Если не задан —
	// MAX шлёт пустой заголовок, проверка проходит.
	secret := os.Getenv("MAX_BOT_API_SECRET")

	api.GetUpdateHandlerFunc(func(update schemes.UpdateInterface) {
		Dispatch(r.Context(), api, update)
		w.WriteHeader(http.StatusOK) // MAX требует 200 OK
	}, secret)(w, r)
}

// Dispatch — общая для webhook и long-polling логика ответов.
func Dispatch(ctx context.Context, api *maxbot.Api, update schemes.UpdateInterface) {
	upd, ok := update.(*schemes.MessageCreatedUpdate)
	if !ok {
		return
	}

	text := strings.TrimSpace(upd.Message.Body.Text)
	reply := startAnswer
	if text != "/start" {
		reply = "Вы написали: " + text
	}

	msg := maxbot.NewMessage().SetText(reply)

	switch {
	case upd.Message.Recipient.ChatId != 0:
		msg.SetChat(upd.Message.Recipient.ChatId)
	case upd.Message.Sender.UserId != 0:
		// Личный диалог: recipient.userId — это сам бот, отвечать надо отправителю.
		msg.SetUser(upd.Message.Sender.UserId)
	default:
		log.Println("Не удалось определить адресата сообщения")
		return
	}

	if err := api.Messages.Send(ctx, msg); err != nil {
		log.Printf("Ошибка отправки: %v", err)
	}
}

// HandleValidate — проверка initData мини-приложения (маршрут /api/validate).
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	json.NewEncoder(w).Encode(validateResponse(req.InitData)) //nolint:errcheck
}

func validateResponse(initData string) map[string]string {
	if initData == "" {
		return map[string]string{"message": "Сервер работает! Откройте приложение внутри MAX."}
	}

	valid, err := ValidateInitData(initData, os.Getenv("MAX_BOT_TOKEN"), maxInitDataAge())
	switch {
	case err != nil && errors.Is(err, ErrExpiredInitData):
		log.Printf("проверка initData не прошла: %v", err)
		return map[string]string{"message": "Данные устарели. Перезапустите приложение."}
	case err != nil:
		log.Printf("проверка initData не прошла: %v", err)
		return map[string]string{"message": "Данные недействительны"}
	case !valid:
		return map[string]string{"message": "Данные недействительны"}
	}
	return map[string]string{"message": "Данные подтверждены! Пользователь аутентифицирован."}
}

// maxInitDataAge — окно жизни initData. По умолчанию 1 час (рекомендация MAX),
// переопределяется MAX_INIT_DATA_MAX_AGE в секундах. <= 0 отключает проверку.
func maxInitDataAge() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MAX_INIT_DATA_MAX_AGE"))
	if raw == "" {
		return time.Hour
	}

	secs, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("MAX_INIT_DATA_MAX_AGE=%q не разобрано, используем 3600", raw)
		return time.Hour
	}
	return time.Duration(secs) * time.Second
}
