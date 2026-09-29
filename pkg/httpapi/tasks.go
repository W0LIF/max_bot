package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"max_bot_api/pkg/storage"
)

const (
	defaultTasksLimit = 100
	maxTasksLimit     = 500
)

type taskCreateReq struct {
	Title     string    `json:"title"`
	Subject   string    `json:"subject"`
	Deadline  time.Time `json:"deadline"`
	Urgent    bool      `json:"urgent"`
	Important bool      `json:"important"`
}

type taskPatchReq struct {
	Title     *string    `json:"title,omitempty"`
	Subject   *string    `json:"subject,omitempty"`
	Deadline  *time.Time `json:"deadline,omitempty"`
	Urgent    *bool      `json:"urgent,omitempty"`
	Important *bool      `json:"important,omitempty"`
	Status    *string    `json:"status,omitempty"`
}

func handleTasksList(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		tasks, err := store.GetTasks(ctx, userID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Не удалось получить задачи")
			return
		}

		if status := r.URL.Query().Get("status"); status != "" {
			if !storage.TaskStatus(status).IsValid() {
				writeErr(w, http.StatusBadRequest, "Недопустимый status")
				return
			}
			filtered := tasks[:0]
			for _, t := range tasks {
				if string(t.Status) == status {
					filtered = append(filtered, t)
				}
			}
			tasks = filtered
		}

		limit := defaultTasksLimit
		if raw := r.URL.Query().Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				writeErr(w, http.StatusBadRequest,
					"limit должен быть положительным числом")
				return
			}
			if n > maxTasksLimit {
				n = maxTasksLimit
			}
			limit = n
		}
		if len(tasks) > limit {
			tasks = tasks[:limit]
		}
		if tasks == nil {
			tasks = []storage.Task{}
		}
		writeJSON(w, http.StatusOK, tasks)
	}
}

func handleTaskCreate(store storage.Store, notifier Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		var req taskCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}

		if err := validateTitle(req.Title); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Deadline.IsZero() {
			writeErr(w, http.StatusBadRequest, "Укажите дедлайн в формате RFC3339")
			return
		}

		task := &storage.Task{
			UserID:    userID,
			Title:     req.Title,
			Subject:   req.Subject,
			Deadline:  req.Deadline.UTC(),
			Urgent:    req.Urgent,
			Important: req.Important,
			Status:    storage.TaskNew,
		}
		id, err := store.CreateTask(ctx, task)
		if err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось сохранить задачу")
			return
		}
		task.ID = id

		notifier.NotifyTaskCreated(ctx, userID, task)

		writeJSON(w, http.StatusCreated, task)
	}
}

func handleTaskPatch(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Некорректный id")
			return
		}

		task, err := store.GetTask(ctx, userID, id)
		if errors.Is(err, storage.ErrTaskNotFound) {
			writeErr(w, http.StatusNotFound, "Задача не найдена")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Ошибка хранилища")
			return
		}

		var req taskPatchReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "Не удалось разобрать тело")
			return
		}

		if req.Title != nil {
			if err := validateTitle(*req.Title); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			task.Title = *req.Title
		}
		if req.Subject != nil {
			task.Subject = *req.Subject
		}
		if req.Deadline != nil {
			if req.Deadline.IsZero() {
				writeErr(w, http.StatusBadRequest, "Дедлайн не может быть пустым")
				return
			}
			task.Deadline = req.Deadline.UTC()
		}
		if req.Urgent != nil {
			task.Urgent = *req.Urgent
		}
		if req.Important != nil {
			task.Important = *req.Important
		}
		if req.Status != nil {
			next := storage.TaskStatus(*req.Status)
			if err := task.CanTransitionTo(next); err != nil {
				if errors.Is(err, storage.ErrInvalidTransition) {
					writeErr(w, http.StatusConflict, err.Error())
					return
				}
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			task.Status = next
		}

		if err := store.UpdateTask(ctx, task); err != nil {
			writeErr(w, http.StatusInternalServerError,
				"Не удалось сохранить задачу")
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

func handleTaskDelete(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := userIDFrom(ctx)

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Некорректный id")
			return
		}

		if err := store.DeleteTask(ctx, userID, id); err != nil {
			if errors.Is(err, storage.ErrTaskNotFound) {
				writeErr(w, http.StatusNotFound, "Задача не найдена")
				return
			}
			writeErr(w, http.StatusInternalServerError, "Не удалось удалить")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateTitle(title string) error {
	n := utf8.RuneCountInString(title)
	if n == 0 {
		return errors.New("Название не может быть пустым")
	}
	if n > 200 {
		return errors.New("Название длиннее 200 символов")
	}
	return nil
}
