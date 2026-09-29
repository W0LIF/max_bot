package httpapi

import (
	"encoding/json"
	"errors"
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

		_, groupErr := store.GetUserGroup(ctx, u.ID)
		if groupErr != nil && !errors.Is(groupErr, storage.ErrGroupNotFound) {
			writeErr(w, http.StatusInternalServerError, "Ошибка хранилища")
			return
		}
		groupSharing := groupErr == nil

		writeJSON(w, http.StatusOK, map[string]any{
			"id":            stored.ID,
			"name":          stored.Name,
			"consent":       stored.Consent,
			"onboarded":     stored.IsOnboarded(),
			"reminders_on":  stored.RemindersOn,
			"group_sharing": groupSharing,
		})
	}
}

func handleConsent(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Value bool `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}
		if err := store.SetConsent(r.Context(), userIDFrom(r.Context()), req.Value); err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				writeErr(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			writeErr(w, http.StatusInternalServerError, "Не удалось сохранить согласие")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func handleReminders(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}
		if err := store.SetReminders(r.Context(), userIDFrom(r.Context()), req.Enabled); err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				writeErr(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			writeErr(w, http.StatusInternalServerError, "Не удалось сохранить настройку")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
