package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"max_bot_api/pkg/storage"
)

type noteCreateReq struct {
	Text   string `json:"text"`
	AllDay bool   `json:"allDay"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

func handleNotesList(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		notes, err := store.GetNotes(ctx, userID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось получить заметки")
			return
		}
		if notes == nil {
			notes = []storage.Note{}
		}
		writeJSON(w, http.StatusOK, notes)
	}
}

func handleNoteCreate(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		var req noteCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}
		if req.Text == "" {
			writeErr(w, http.StatusBadRequest, "text обязателен")
			return
		}

		start, err := parseTime(req.Start)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "start: "+err.Error())
			return
		}
		end, err := parseTime(req.End)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "end: "+err.Error())
			return
		}
		if end.Before(start) {
			writeErr(w, http.StatusBadRequest, "end не может быть раньше start")
			return
		}

		n := &storage.Note{
			UserID: userID,
			Text:   req.Text,
			AllDay: req.AllDay,
			Start:  start,
			End:    end,
		}
		id, err := store.CreateNote(ctx, n)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось сохранить заметку")
			return
		}
		n.ID = id
		writeJSON(w, http.StatusCreated, n)
	}
}

func handleNoteDelete(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Некорректный id")
			return
		}
		if err := store.DeleteNote(ctx, userID, id); err != nil {
			if errors.Is(err, storage.ErrNoteNotFound) {
				writeErr(w, http.StatusNotFound, "Заметка не найдена")
				return
			}
			writeErr(w, http.StatusInternalServerError, "Не удалось удалить")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("пустая дата")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, errors.New("ожидали RFC3339")
	}
	return t.UTC(), nil
}
