package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"max_bot_api/pkg/storage"
)

type moodCreateReq struct {
	Mood string `json:"mood"`
	Note string `json:"note,omitempty"`
}

func handleMoodsList(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		days := 7
		if raw := r.URL.Query().Get("days"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				writeErr(w, http.StatusBadRequest,
					"days должен быть положительным числом")
				return
			}
			days = n
		}

		moods, err := store.GetMoods(ctx, userID, days)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось получить настроения")
			return
		}
		if moods == nil {
			moods = []storage.Mood{}
		}
		writeJSON(w, http.StatusOK, moods)
	}
}

func handleMoodCreate(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		var req moodCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}

		value := storage.MoodValue(req.Mood)
		if !value.IsValid() {
			writeErr(w, http.StatusBadRequest, "mood должен быть good|ok|bad")
			return
		}

		m := &storage.Mood{
			UserID:    userID,
			Value:     value,
			Note:      req.Note,
			CreatedAt: time.Now().UTC(),
		}
		id, err := store.CreateMood(ctx, m)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось сохранить настроение")
			return
		}
		m.ID = id
		writeJSON(w, http.StatusCreated, m)
	}
}
