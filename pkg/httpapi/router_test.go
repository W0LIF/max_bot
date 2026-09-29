package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"max_bot_api/pkg/storage"
)

const testToken = "123456:TEST_TOKEN_ABCDEF"

// --- helpers ---

// signInitData — локальная реализация подписи MAX. Специально не
// используем pkg/bot, чтобы тесты httpapi проверяли контракт, а не
// зависели от внутренностей bot.
func signInitData(t *testing.T, token string, userID int64, authDate time.Time) string {
	t.Helper()

	user := `{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"Тест"}`
	form := url.Values{
		"auth_date": {strconv.FormatInt(authDate.Unix(), 10)},
		"user":      {user},
	}
	form.Set("hash", signLaunch(t, token, form))
	return form.Encode()
}

func signLaunch(t *testing.T, token string, form url.Values) string {
	t.Helper()

	keys := make([]string, 0, len(form))
	for k := range form {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var launch string
	for i, k := range keys {
		if i > 0 {
			launch += "\n"
		}
		launch += k + "=" + form.Get(k)
	}

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(token))
	signer := hmac.New(sha256.New, mac.Sum(nil))
	signer.Write([]byte(launch))
	return hex.EncodeToString(signer.Sum(nil))
}

// newTestRouter поднимает роутер на MemoryStore с валидным initData.
// Возвращает роутер, initData и стор (для проверок состояния).
func newTestRouter(t *testing.T) (http.Handler, string, storage.Store) {
	t.Helper()
	t.Setenv("MAX_BOT_TOKEN", testToken)
	t.Setenv("MAX_INIT_DATA_MAX_AGE", "3600")

	store := storage.NewMemoryStore()
	router := New(store, nil)
	initData := signInitData(t, testToken, 42, time.Now())

	return router, initData, store
}

// do — короткий хелпер: запрос + запись ответа.
func do(router http.Handler, method, path, initData, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if initData != "" {
		r.Header.Set("X-Max-Init-Data", initData)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, r)
	return rec
}

// --- авторизация ---

func TestRouter_NoHeader_Unauthorized(t *testing.T) {
	router, _, _ := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/me", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ожидали 401 без заголовка, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
}

func TestRouter_BadInitData_Unauthorized(t *testing.T) {
	router, _, _ := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/me",
		"auth_date=1&user=%7B%7D&hash=deadbeef", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ожидали 401 с битым initData, получили %d", rec.Code)
	}
}

func TestRouter_ValidInitData_OK(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/me", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
}

// --- /api/me ---

func TestRouter_Me_CreatesUser(t *testing.T) {
	router, initData, store := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/me", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":42`) {
		t.Fatalf("нет id=42 в ответе: %s", rec.Body.String())
	}

	// Проверяем, что пользователь реально создан.
	u, err := store.GetUser(t.Context(), 42)
	if err != nil {
		t.Fatalf("пользователь не создан: %v", err)
	}
	if u.Name != "Тест" {
		t.Fatalf("имя не совпадает: %q", u.Name)
	}
}

func TestRouter_Me_ExistingUser(t *testing.T) {
	router, initData, store := newTestRouter(t)

	// Первое обращение — создаёт.
	_ = do(router, http.MethodGet, "/api/me", initData, "")

	// Ставим consent — на следующем запросе должен быть true.
	if err := store.SetConsent(t.Context(), 42, true); err != nil {
		t.Fatalf("SetConsent: %v", err)
	}

	rec := do(router, http.MethodGet, "/api/me", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"consent":true`) {
		t.Fatalf("consent не true: %s", rec.Body.String())
	}
}

func TestRouter_ConsentAndReminders(t *testing.T) {
	router, initData, store := newTestRouter(t)
	_ = do(router, http.MethodGet, "/api/me", initData, "")

	rec := do(router, http.MethodPost, "/api/consent", initData, `{"value":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/consent: ожидали 200, получили %d, body=%s", rec.Code, rec.Body.String())
	}
	rec = do(router, http.MethodPost, "/api/me/reminders", initData, `{"enabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/me/reminders: ожидали 200, получили %d, body=%s", rec.Code, rec.Body.String())
	}

	user, err := store.GetUser(t.Context(), 42)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if !user.Consent || !user.RemindersOn {
		t.Fatalf("ожидали consent и reminders_on true, получили %+v", user)
	}
}

func TestRouter_Me_AutoGroup(t *testing.T) {
	router, initData, store := newTestRouter(t)

	_ = do(router, http.MethodGet, "/api/me", initData, "")

	// После /api/me пользователь должен быть в демо-группе.
	gid, err := store.GetUserGroup(t.Context(), 42)
	if err != nil {
		t.Fatalf("пользователь не в группе: %v", err)
	}
	if gid != storage.DemoGroupID {
		t.Fatalf("ожидали группу %d, получили %d", storage.DemoGroupID, gid)
	}
}

// --- CRUD задач ---

func TestRouter_TaskCreateAndList(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	body := `{"title":"Сдать лабу","subject":"Информатика",` +
		`"deadline":"2030-01-01T12:00:00Z","urgent":true,"important":true}`

	rec := do(router, http.MethodPost, "/api/tasks", initData, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: ожидали 201, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}

	rec = do(router, http.MethodGet, "/api/tasks", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Сдать лабу") {
		t.Fatalf("нет задачи в списке: %s", rec.Body.String())
	}
}

func TestRouter_TaskCreate_EmptyTitle(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"","deadline":"2030-01-01T12:00:00Z"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

func TestRouter_TaskCreate_MissingDeadline(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"А"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

func TestRouter_TaskList_FilterByStatus(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	// Две задачи: new и done.
	_ = do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"новая","deadline":"2030-01-01T12:00:00Z"}`)
	_ = do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"выполненная","deadline":"2030-01-01T12:00:00Z"}`)

	// Переводим вторую в done.
	_ = do(router, http.MethodPatch, "/api/tasks/2", initData,
		`{"status":"done"}`)

	rec := do(router, http.MethodGet, "/api/tasks?status=done", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "выполненная") {
		t.Fatalf("нет выполненной задачи: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "новая") {
		t.Fatalf("вернулась активная задача: %s", rec.Body.String())
	}
}

func TestRouter_TaskList_FilterByStatus_Invalid(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/tasks?status=bogus", initData, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

func TestRouter_TaskList_Limit(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	// Создаём 3 задачи.
	for i := 1; i <= 3; i++ {
		_ = do(router, http.MethodPost, "/api/tasks", initData,
			`{"title":"задача","deadline":"2030-01-01T12:00:00Z"}`)
	}

	rec := do(router, http.MethodGet, "/api/tasks?limit=2", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	// Считаем количество "id" в JSON — грубо, но для теста достаточно.
	if n := strings.Count(rec.Body.String(), `"id":`); n != 2 {
		t.Fatalf("ожидали 2 задачи, получили %d: %s", n, rec.Body.String())
	}
}

func TestRouter_TaskList_Limit_Invalid(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodGet, "/api/tasks?limit=0", initData, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

func TestRouter_TaskPatch_Partial(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	_ = do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"старое","deadline":"2030-01-01T12:00:00Z"}`)

	rec := do(router, http.MethodPatch, "/api/tasks/1", initData,
		`{"title":"новое"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "новое") {
		t.Fatalf("название не изменилось: %s", rec.Body.String())
	}
}

func TestRouter_TaskPatch_InvalidTransition(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	_ = do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"A","deadline":"2030-01-01T12:00:00Z"}`)

	// new → done — разрешено.
	rec := do(router, http.MethodPatch, "/api/tasks/1", initData,
		`{"status":"done"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("new→done: ожидали 200, получили %d", rec.Code)
	}

	// done → in_progress — запрещено → 409.
	rec = do(router, http.MethodPatch, "/api/tasks/1", initData,
		`{"status":"in_progress"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("done→in_progress: ожидали 409, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
}

func TestRouter_TaskDelete(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	_ = do(router, http.MethodPost, "/api/tasks", initData,
		`{"title":"удалить","deadline":"2030-01-01T12:00:00Z"}`)

	rec := do(router, http.MethodDelete, "/api/tasks/1", initData, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ожидали 204, получили %d", rec.Code)
	}

	// Повторное удаление → 404.
	rec = do(router, http.MethodDelete, "/api/tasks/1", initData, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("ожидали 404, получили %d", rec.Code)
	}
}

func TestRouter_TaskForeignUser_NotFound(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	// Создаём задачу от другого пользователя вручную.
	store := storage.NewMemoryStore()
	_ = store // на самом деле используется тот же стор; создаём вручную:

	// Извлечём стор из роутера сложно, поэтому просто создадим задачу
	// от userID=99 напрямую через /api/me другого initData.
	otherInit := signInitData(t, testToken, 99, time.Now())

	_ = do(router, http.MethodPost, "/api/tasks", otherInit,
		`{"title":"чужая","deadline":"2030-01-01T12:00:00Z"}`)

	// Пытаемся получить её от userID=42 → 404.
	rec := do(router, http.MethodDelete, "/api/tasks/1", initData, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("чужая задача: ожидали 404, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
}

// --- Moods ---

func TestRouter_MoodCreateAndList(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/moods", initData,
		`{"mood":"good","note":"ок"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST mood: ожидали 201, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}

	rec = do(router, http.MethodGet, "/api/moods?days=7", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET moods: ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"value":"good"`) {
		t.Fatalf("нет записи good: %s", rec.Body.String())
	}
}

func TestRouter_MoodCreate_InvalidValue(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/moods", initData,
		`{"mood":"bogus"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

// --- Notes ---

func TestRouter_NoteCreateAndDelete(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/notes", initData,
		`{"text":"встреча","start":"2030-01-01T12:00:00Z","end":"2030-01-01T13:00:00Z"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST note: ожидали 201, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}

	rec = do(router, http.MethodGet, "/api/notes", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET notes: ожидали 200, получили %d", rec.Code)
	}

	rec = do(router, http.MethodDelete, "/api/notes/1", initData, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE note: ожидали 204, получили %d", rec.Code)
	}
}

func TestRouter_NoteCreate_EmptyText(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/notes", initData,
		`{"text":"","start":"2030-01-01T12:00:00Z","end":"2030-01-01T13:00:00Z"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

// --- Feedback ---

func TestRouter_Feedback(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/feedback", initData,
		`{"text":"всё работает"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("нет ok:true: %s", rec.Body.String())
	}
}

// --- Group ---

func TestRouter_GroupMembers_EmptyBeforeMe(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	// Без предварительного /api/me пользователь не в группе — пустой список.
	rec := do(router, http.MethodGet, "/api/group/members", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("ожидали пустой массив, получили %s", rec.Body.String())
	}
}

func TestRouter_GroupMembers_AfterMe(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	// Прогреваем /api/me — пользователь попадает в демо-группу.
	_ = do(router, http.MethodGet, "/api/me", initData, "")

	rec := do(router, http.MethodGet, "/api/group/members", initData, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":42`) {
		t.Fatalf("нет пользователя в группе: %s", rec.Body.String())
	}
}

// --- Scan ---

func TestRouter_Scan_RequiresMultipart(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/tasks/scan", initData, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400 без multipart, получили %d", rec.Code)
	}
}

// --- Auth ---

func TestAuthValidate_Success(t *testing.T) {
	router, initData, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/auth/validate", "",
		`{"initData":`+strconv.Quote(initData)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d, body=%s",
			rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("нет ok:true: %s", rec.Body.String())
	}
}

func TestAuthValidate_BadInitData(t *testing.T) {
	router, _, _ := newTestRouter(t)

	rec := do(router, http.MethodPost, "/api/auth/validate", "",
		`{"initData":"garbage"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ожидали 401, получили %d", rec.Code)
	}
}

// --- CORS ---

func TestRouter_CORS_Options(t *testing.T) {
	router, _, _ := newTestRouter(t)

	rec := do(router, http.MethodOptions, "/api/me", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("OPTIONS: ожидали 200, получили %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("нет CORS-заголовка")
	}
}
