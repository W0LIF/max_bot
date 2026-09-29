package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"max_bot_api/pkg/bot"
)

// RequireInitData проверяет заголовок X-Max-Init-Data и кладёт user_id
// и профиль в context. На ErrExpiredInitData — отдельный текст 401.
func RequireInitData(token string, maxAge time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-Max-Init-Data")
		if raw == "" {
			writeErr(w, http.StatusUnauthorized, "Не авторизован")
			return
		}

		u, err := bot.ValidateAndParseInitData(raw, token, maxAge, time.Now())
		switch {
		case errors.Is(err, bot.ErrExpiredInitData):
			writeErr(w, http.StatusUnauthorized,
				"Данные устарели, перезапустите приложение")
			return
		case err != nil:
			writeErr(w, http.StatusUnauthorized, "Данные недействительны")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, u.ID)
		ctx = context.WithValue(ctx, ctxKeyUser, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
