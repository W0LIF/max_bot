// cmd/server — HTTP API поверх общего SQLite/Turso-хранилища.
//
//	go run ./cmd/server
//
// Порт — 8080 (совпадает с прокси Vite: frontend/vite.config.ts).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/httpapi"
	"max_bot_api/pkg/storage"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения системы")
	}

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

	notifier := bot.NewNotifier()
	handler := httpapi.New(store, notifier)

	addr := ":8080"
	log.Printf("HTTP API запущен на %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
