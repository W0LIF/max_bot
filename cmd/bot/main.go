package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/storage"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения системы")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	api, err := bot.NewAPIClient(os.Getenv("MAX_BOT_TOKEN"))
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

	store := storage.NewMemoryStore()
	b := bot.New(api, store)

	for update := range api.GetUpdates(ctx) {
		b.Dispatch(ctx, update)
	}
}
