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
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/reminders"
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
	fmt.Printf("Бот запущен: %s (ID: %d, Username: %s)\n",
		botInfo.Name, botInfo.UserId, botInfo.Username)

	// --- Хранилище ---
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "bot.db"
	}
	store, err := storage.NewConfiguredStore(dbPath)
	if err != nil {
		log.Fatalf("Не удалось открыть БД %q: %v", dbPath, err)
	}
	defer store.Close()
	log.Println("Хранилище готово")

	// --- Шедулер напоминаний ---
	remindersOut := make(chan storage.Task, 100)
	scheduler := reminders.NewScheduler(
		store,
		remindersOut,
		15*time.Minute,
		24*time.Hour,
		slog.Default(),
	)
	go scheduler.Run(ctx)
	log.Println("Шедулер напоминаний запущен")

	// --- Читатель канала напоминаний (B6) ---
	//
	// Без этого цикла scheduler.Run блокируется на отправке в канал,
	// и напоминания не уходят ни разу. Фильтр reminders_on — здесь,
	// сам шедулер его не смотрит.
	go runRemindersConsumer(ctx, store, api, remindersOut)
	log.Println("Читатель напоминаний запущен")

	// --- Ошибки MAX API ---
	go func() {
		for errMessage := range api.GetErrors() {
			log.Println(errMessage)
		}
	}()

	// --- Бот ---
	b := bot.New(api, store)

	for update := range api.GetUpdates(ctx) {
		b.Dispatch(ctx, update)
	}
}
