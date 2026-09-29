package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"max_bot_api/pkg/storage"
)

func handleGroupSharing(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}

		var err error
		if req.Enabled {
			err = store.EnsureGroup(r.Context(), userIDFrom(r.Context()), storage.DemoGroupID)
		} else {
			err = store.LeaveGroups(r.Context(), userIDFrom(r.Context()))
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Не удалось изменить доступ к задачам")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
	}
}

func handleGroupMembers(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		gid, err := store.GetUserGroup(ctx, userID)
		if errors.Is(err, storage.ErrGroupNotFound) {
			// Пользователь ещё не в группе — пустой список, не 404,
			// чтобы фронт не падал.
			writeJSON(w, http.StatusOK, []storage.Member{})
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Ошибка хранилища")
			return
		}

		members, err := store.GetGroupMembers(ctx, gid)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось получить участников")
			return
		}
		if members == nil {
			members = []storage.Member{}
		}
		writeJSON(w, http.StatusOK, members)
	}
}

func handleGroupTasks(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		gid, err := store.GetUserGroup(ctx, userID)
		if errors.Is(err, storage.ErrGroupNotFound) {
			writeJSON(w, http.StatusOK, []storage.Task{})
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Ошибка хранилища")
			return
		}

		members, err := store.GetGroupMembers(ctx, gid)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось получить участников")
			return
		}

		var out []storage.Task
		for _, m := range members {
			tasks, err := store.GetTasks(ctx, m.ID)
			if err != nil {
				continue
			}
			for _, t := range tasks {
				t.Group = true
				out = append(out, t)
			}
		}
		if out == nil {
			out = []storage.Task{}
		}
		writeJSON(w, http.StatusOK, out)
	}
}
