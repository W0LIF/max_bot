// cmd/bot — локальный запуск бота на long-polling (GetUpdates).
//
// Это обычный демон с бесконечным циклом, он НЕ может жить в api/:
// Vercel считает любой .go-файл там serverless-функцией и требует
// func Handler вместо func main.
//
// Запуск локально:
//
//	go run ./cmd/bot
//
// Для Vercel используется вебхук /api/webhook — см. api/webhook.go.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	maxbot "github.com/max-messenger/max-bot-api-client-go"

	"max_bot_api/pkg/bot"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения системы")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	api, err := maxbot.New(os.Getenv("MAX_BOT_TOKEN"))
	if err != nil {
		log.Fatalf("Ошибка инициализации: %v", err)
	}

	botInfo, err := api.Bots.GetBot(ctx)
	if err != nil {
		log.Fatalf("Не удалось получить информацию о боте: %v", err)
	}
	fmt.Printf("Бот запущен: %s (ID: %d, Username: %s)\n", botInfo.Name, botInfo.UserId, botInfo.Username)

	go func() {
		for errMessage := range api.GetErrors() {
			log.Println(errMessage)
		}
	}()

	for update := range api.GetUpdates(ctx) {
		bot.Dispatch(ctx, api, update)
	}
}
