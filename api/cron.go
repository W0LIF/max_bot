package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/reminders"
	"max_bot_api/pkg/storage"
)

// Cron dispatches due reminders from Vercel Cron. Configure CRON_SECRET in
// the Vercel project; Cron sends it as an Authorization Bearer token.
func Cron(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	secret := os.Getenv("CRON_SECRET")
	if secret == "" {
		http.Error(w, "Cron is not configured", http.StatusServiceUnavailable)
		return
	}
	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(provided) != len(secret) || subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	store, err := getStore()
	if err != nil {
		http.Error(w, "Storage unavailable", http.StatusInternalServerError)
		return
	}
	client, err := bot.NewAPIClient(os.Getenv("MAX_BOT_TOKEN"))
	if err != nil {
		log.Printf("cron: initialize MAX client: %v", err)
		http.Error(w, "Bot API unavailable", http.StatusInternalServerError)
		return
	}

	// Hobby Cron runs once per day with up to 59 minutes of timing variance.
	sent, err := reminders.DispatchDue(r.Context(), store, time.Now().Add(25*time.Hour), func(ctx context.Context, task storage.Task) error {
		return bot.SendTaskReminder(ctx, client, task)
	})
	if err != nil {
		log.Printf("cron: dispatch reminders: %v", err)
		http.Error(w, "Reminder dispatch failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"sent": sent})
}
