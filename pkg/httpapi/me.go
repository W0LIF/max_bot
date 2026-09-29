package httpapi

import (
	"errors"
	"log"
	"net/http"

	"max_bot_api/pkg/storage"
)

func handleMe(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		u := userFrom(ctx)

		stored, err := store.GetUser(ctx, u.ID)
		if errors.Is(err, storage.ErrUserNotFound) {
			// Первое обращение — создаём запись.
			newUser := &storage.User{
				ID:   u.ID,
				Name: u.FirstName,
			}
			if err := store.SaveUser(ctx, newUser); err != nil {
				writeErr(w, http.StatusInternalServerError,
					"Не удалось создать пользователя")
				return
			}
			stored = newUser
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, "Ошибка хранилища")
			return
		}

		// Автоматически добавляем в демо-группу, если ещё не там.
		// Ошибку логируем, но не валим запрос — /api/me должен
		// отвечать даже если группа недоступна.
		if _, err := store.GetUserGroup(ctx, u.ID); errors.Is(err, storage.ErrGroupNotFound) {
			if err := store.EnsureGroup(ctx, u.ID, storage.DemoGroupID); err != nil {
				log.Printf("me: EnsureGroup(%d): %v", u.ID, err)
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":           stored.ID,
			"name":         stored.Name,
			"consent":      stored.Consent,
			"onboarded":    stored.IsOnboarded(),
			"reminders_on": stored.RemindersOn,
		})
	}
}
