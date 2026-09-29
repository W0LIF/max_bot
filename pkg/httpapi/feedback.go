package httpapi

import (
	"encoding/json"
	"net/http"

	"max_bot_api/pkg/storage"
)

type feedbackReq struct {
	Text string `json:"text"`
}

func handleFeedback(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		var req feedbackReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}
		if req.Text == "" {
			writeErr(w, http.StatusBadRequest, "text обязателен")
			return
		}

		f := &storage.Feedback{UserID: userID, Text: req.Text}
		if _, err := store.CreateFeedback(ctx, f); err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось сохранить отзыв")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
