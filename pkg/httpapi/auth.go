package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"max_bot_api/pkg/bot"
)

type authValidateReq struct {
	InitData string `json:"initData"`
}

func handleAuthValidate(token string, maxAge time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authValidateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}

		u, err := bot.ValidateAndParseInitData(req.InitData, token, maxAge, time.Now())
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "Данные недействительны")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"user": map[string]any{
				"id":   u.ID,
				"name": u.FirstName,
			},
		})
	}
}
