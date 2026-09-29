package httpapi

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/storage"
)

// Notifier — узкий интерфейс, чтобы httpapi не зависел от pkg/bot
// в обратную сторону (иначе цикл httpapi → bot → httpapi).
type Notifier interface {
	NotifyTaskCreated(ctx context.Context, userID int64, task *storage.Task)
}

type nopNotifier struct{}

func (nopNotifier) NotifyTaskCreated(context.Context, int64, *storage.Task) {}

// New собирает HTTP-роутер поверх Store. notifier может быть nil —
// тогда используется заглушка (ничего не делает).
//
// Публичные маршруты (без X-Max-Init-Data):
//
//	POST /api/auth/validate — вход мини-приложения
//	POST /api/webhook       — апдейты от MAX
//	POST /api/validate      — legacy-отладка
//
// Защищённые (нужен X-Max-Init-Data) — регистрируются в отдельном
// mux и оборачиваются RequireInitData. ServeMux выбирает самый
// специфичный паттерн, поэтому точные "/api/auth/validate" и
// "/api/webhook" не попадают под middleware, хотя формально
// подходят под "/api/".
func New(store storage.Store, notifier Notifier) http.Handler {
	if notifier == nil {
		notifier = nopNotifier{}
	}

	token := os.Getenv("MAX_BOT_TOKEN")
	maxAge := initDataMaxAge()

	mux := http.NewServeMux()

	// --- Публичные ---
	mux.HandleFunc("POST /api/auth/validate", handleAuthValidate(token, maxAge))
	mux.HandleFunc("POST /api/webhook", bot.HandleWebhook(store))
	mux.HandleFunc("POST /api/validate", bot.HandleValidate) // legacy

	// --- Защищённые ---
	protected := http.NewServeMux()

	protected.HandleFunc("GET /api/me", handleMe(store))
	protected.HandleFunc("POST /api/consent", handleConsent(store))
	protected.HandleFunc("POST /api/me/reminders", handleReminders(store))

	protected.HandleFunc("GET /api/tasks", handleTasksList(store))
	protected.HandleFunc("POST /api/tasks", handleTaskCreate(store, notifier))
	protected.HandleFunc("PATCH /api/tasks/{id}", handleTaskPatch(store))
	protected.HandleFunc("DELETE /api/tasks/{id}", handleTaskDelete(store))

	protected.HandleFunc("GET /api/moods", handleMoodsList(store))
	protected.HandleFunc("POST /api/moods", handleMoodCreate(store))

	protected.HandleFunc("GET /api/notes", handleNotesList(store))
	protected.HandleFunc("POST /api/notes", handleNoteCreate(store))
	protected.HandleFunc("DELETE /api/notes/{id}", handleNoteDelete(store))

	protected.HandleFunc("GET /api/group/members", handleGroupMembers(store))
	protected.HandleFunc("GET /api/group/tasks", handleGroupTasks(store))

	protected.HandleFunc("POST /api/tasks/scan", handleScan())
	protected.HandleFunc("POST /api/feedback", handleFeedback(store))

	// Всё, что под /api/, уходит в middleware. Точные паттерны выше
	// (зарегистрированные прямо в mux) имеют приоритет в ServeMux
	// Go 1.22+, поэтому публичные маршруты не перехватываются.
	mux.Handle("/api/", RequireInitData(token, maxAge, protected))

	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, X-Max-Init-Data")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// initDataMaxAge — окно жизни initData из MAX_INIT_DATA_MAX_AGE
// (в секундах). Пустое или невалидное значение → 1 час.
func initDataMaxAge() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MAX_INIT_DATA_MAX_AGE"))
	if raw == "" {
		return time.Hour
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		return time.Hour
	}
	return time.Duration(secs) * time.Second
}
