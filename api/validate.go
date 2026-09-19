package handler

import (
	"encoding/json"
	"net/http"
	"os"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// CORS для локальной разработки (на Vercel не помешает)
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
			"message": "Сервер работает! Откройте приложение внутри MAX.",
		})
		return
	}

	valid, err := ValidateInitData(req.InitData, os.Getenv("MAX_BOT_TOKEN"))
	if err != nil || !valid {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Данные недействительны",
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Данные подтверждены! Пользователь аутентифицирован.",
	})
}
