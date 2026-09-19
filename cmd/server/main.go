// cmd/server — локальный HTTP-сервер для разработки фронтенда.
//
// Поднимает те же обработчики, что и Vercel-функции в api/, чтобы Vite
// мог проксировать на него /api (см. frontend/vite.config.ts).
//
// Запуск:
//
//	go run ./cmd/server
//
// На проде этот бинарь не нужен — там работают api/webhook.go и
// api/validate.go как serverless-функции.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"max_bot_api/pkg/bot"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения системы")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/webhook", bot.HandleWebhook)
	mux.HandleFunc("/api/validate", bot.HandleValidate)

	addr := ":" + envOr("PORT", "8080")
	log.Printf("HTTP-сервер запущен на %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка HTTP-сервера: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
