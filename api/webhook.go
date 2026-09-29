package handler

import (
	"log"
	"net/http"
	"os"
	"sync"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/storage"
)

var (
	storeOnce sync.Once
	store     storage.Store
	storeErr  error
)

// getStore returns one configured store per warm Vercel instance. Set
// TURSO_DATABASE_URL and TURSO_AUTH_TOKEN for durable storage; the /tmp
// SQLite fallback is only suitable for local or disposable deployments.
func getStore() (storage.Store, error) {
	storeOnce.Do(func() {
		path := os.Getenv("DB_PATH")
		if path == "" {
			path = "/tmp/bot.db" // Vercel: единственная writable-директория
		}
		if os.Getenv("TURSO_DATABASE_URL") == "" {
			log.Printf("webhook: TURSO_DATABASE_URL не задан, данные в %s будут временными", path)
		}
		store, storeErr = storage.NewConfiguredStore(path)
		if storeErr != nil {
			log.Printf("webhook: не удалось открыть хранилище: %v", storeErr)
		}
	})
	return store, storeErr
}

// Webhook — точка входа serverless-функции Vercel.
func Webhook(w http.ResponseWriter, r *http.Request) {
	st, err := getStore()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	bot.HandleWebhook(st)(w, r)
}
