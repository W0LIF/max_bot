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

// getStore возвращает singleton-стор для текущего процесса.
// На Vercel процесс живёт между вызовами в рамках «тёплого» инстанса,
// поэтому sync.Once даёт переиспользование. При холодном старте —
// открывается заново.
func getStore() (storage.Store, error) {
	storeOnce.Do(func() {
		path := os.Getenv("DB_PATH")
		if path == "" {
			path = "/tmp/bot.db" // Vercel: единственная writable-директория
		}
		store, storeErr = storage.NewSQLiteStore(path)
		if storeErr != nil {
			log.Printf("webhook: не удалось открыть БД %q: %v", path, storeErr)
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
