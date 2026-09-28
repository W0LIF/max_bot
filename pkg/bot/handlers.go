// Package bot содержит общую логику бота MAX: обработку апдейтов,
// ответы на сообщения и проверку подписи мини-приложения.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"

	"max_bot_api/pkg/storage"
)

// --- Точка входа ---

// Dispatch — общая точка входа для long-polling и webhook.
func (b *Bot) Dispatch(ctx context.Context, update schemes.UpdateInterface) {
	switch u := update.(type) {
	case *schemes.MessageCreatedUpdate:
		b.onMessage(ctx, u)
	case *schemes.MessageCallbackUpdate:
		b.onCallback(ctx, u)
	default:
		// прочие апдейты (bot_started, удаления и т.п.) пока игнорируем
	}
}

// --- Обработка текстовых сообщений ---

func (b *Bot) onMessage(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	userID := upd.Message.Sender.UserId
	text := strings.TrimSpace(upd.Message.Body.Text)

	if text == "/start" || strings.EqualFold(text, "start") {
		b.handleStart(ctx, upd)
		return
	}

	switch b.fsm.Get(userID) {
	case StateAwaitTaskTitle:
		b.onTaskTitle(ctx, upd, text)
	case StateAwaitTaskDeadline:
		b.onTaskDeadline(ctx, upd, text)
	case StateAwaitTaskPriority:
		b.reply(ctx, upd, "Выбери приоритет кнопкой ниже 👇", kbPriority())
	case StateAwaitEditTitle:
		b.onEditTitle(ctx, upd, text)
	case StateAwaitEditDeadline:
		b.onEditDeadline(ctx, upd, text)
	case StateAwaitEditPriority:
		b.reply(ctx, upd, "Выбери приоритет кнопкой ниже 👇", kbPriority())
	default:
		b.fallback(ctx, upd)
	}
}

// handleStart — приветствие и онбординг.
func (b *Bot) handleStart(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	userID := upd.Message.Sender.UserId

	// Если пользователь уже в хранилище — сразу меню.
	if _, err := b.store.GetUser(ctx, userID); err == nil {
		b.reply(ctx, upd, "С возвращением! Что делаем?", kbMainMenu())
		return
	}

	// Нового — сохраняем, иначе SetConsent/SetReminders не найдут его.
	if err := b.store.SaveUser(ctx, &storage.User{
		ID:          userID,
		OnboardedAt: time.Now(),
	}); err != nil {
		log.Printf("SaveUser: %v", err)
	}

	b.reply(ctx, upd,
		"Привет! Я помогу следить за дедлайнами и не выгореть.\n\n"+
			"Согласны на обработку данных?",
		kbConsent())
}

func (b *Bot) fallback(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	b.reply(ctx, upd, "Не понял. Вот меню:", kbMainMenu())
}

// --- Обработка нажатий на кнопки ---

func (b *Bot) onCallback(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	userID := upd.Callback.User.UserId
	payload := upd.Callback.Payload

	switch payload {
	case CbConsentYes:
		if err := b.store.SetConsent(ctx, userID, true); err != nil {
			log.Printf("SetConsent: %v", err)
		}
		b.sendToUser(ctx, userID,
			"Отлично! Присылать напоминания о дедлайнах?", kbReminders())

	case CbConsentNo:
		if err := b.store.SetConsent(ctx, userID, false); err != nil {
			log.Printf("SetConsent: %v", err)
		}
		b.sendToUser(ctx, userID,
			"Понял. Работаем в ограниченном режиме — персональные данные не сохраняем.", nil)
		b.sendToUser(ctx, userID, "Что делаем?", kbMainMenu())

	case CbRemindYes:
		_ = b.store.SetReminders(ctx, userID, true)
		b.sendToUser(ctx, userID, "Готово! Буду напоминать о дедлайнах.", kbMainMenu())

	case CbRemindNo:
		_ = b.store.SetReminders(ctx, userID, false)
		b.sendToUser(ctx, userID, "Ок, без напоминаний.", kbMainMenu())

	case CbMenuAdd:
		b.fsm.Set(userID, StateAwaitTaskTitle)
		b.sendToUser(ctx, userID, "Как назовём задачу?", nil)

	case CbMenuList:
		b.showTaskList(ctx, upd)
	case CbMenuEdit:
		b.showEditList(ctx, upd)
	case CbMenuDelete:
		b.showDeleteList(ctx, upd)
	case CbMenuDone:
		b.showDoneList(ctx, upd)

	case CbMoodGood, CbMoodOK, CbMoodBad:
		b.onMood(ctx, upd)

	default:
		action, id, ok := parseCallback(payload)
		if !ok {
			b.sendToUser(ctx, userID, "Кнопка устарела. Вот меню:", kbMainMenu())
			return
		}
		switch action {
		case "priority":
			switch b.fsm.Get(userID) {
			case StateAwaitTaskPriority:
				b.onPriority(ctx, upd, id)
			case StateAwaitEditPriority:
				b.onEditPriority(ctx, upd, id)
			default:
				b.sendToUser(ctx, userID, "Кнопка устарела. Вот меню:", kbMainMenu())
			}
		case "edit:title":
			b.onEditFieldTitle(ctx, upd)
		case "edit:deadline":
			b.onEditFieldDeadline(ctx, upd)
		case "edit:priority":
			b.onEditFieldPriority(ctx, upd)
		case "task:done":
			b.onTaskDone(ctx, upd, id)
		case "task:edit":
			b.onTaskEdit(ctx, upd, id)
		case "task:delete":
			b.onTaskDelete(ctx, upd, id)
		case "task:delconfirm":
			b.onTaskDeleteConfirm(ctx, upd, id)
		default:
			b.sendToUser(ctx, userID, "Кнопка устарела. Вот меню:", kbMainMenu())
		}
	}
}

// --- Отправка сообщений ---

func (b *Bot) reply(ctx context.Context, upd *schemes.MessageCreatedUpdate, text string, kb *maxbot.Keyboard) {
	if upd.Message.Recipient.ChatId != 0 {
		b.sendToChat(ctx, upd.Message.Recipient.ChatId, text, kb)
		return
	}
	b.sendToUser(ctx, upd.Message.Sender.UserId, text, kb)
}

func (b *Bot) sendToUser(ctx context.Context, userID int64, text string, kb *maxbot.Keyboard) {
	msg := maxbot.NewMessage().SetText(text).SetUser(userID)
	if kb != nil {
		msg.AddKeyboard(kb)
	}
	if err := b.api.Messages.Send(ctx, msg); err != nil {
		log.Printf("send to user %d error: %v", userID, err)
	}
}

func (b *Bot) sendToChat(ctx context.Context, chatID int64, text string, kb *maxbot.Keyboard) {
	msg := maxbot.NewMessage().SetText(text).SetChat(chatID)
	if kb != nil {
		msg.AddKeyboard(kb)
	}
	if err := b.api.Messages.Send(ctx, msg); err != nil {
		log.Printf("send to chat %d error: %v", chatID, err)
	}
}

// --- Webhook ---

// HandleWebhook — обработчик webhook от MAX (маршрут /api/webhook).
//
// Внимание: на Vercel каждый запрос создаёт новый Store в памяти, поэтому
// онбординг между запросами не сохраняется. Общая задача №4 из распределения.
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		log.Println("MAX_BOT_TOKEN не задан")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	api, err := NewAPIClient(token)
	if err != nil {
		log.Printf("Ошибка инициализации бота: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	b := New(api, storage.NewMemoryStore())

	secret := os.Getenv("MAX_BOT_API_SECRET")

	api.GetUpdateHandlerFunc(func(update schemes.UpdateInterface) {
		b.Dispatch(r.Context(), update)
		w.WriteHeader(http.StatusOK)
	}, secret)(w, r)
}

// --- Проверка initData мини-приложения (маршрут /api/validate) ---

func HandleValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InitData string `json:"initData"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validateResponse(req.InitData)) //nolint:errcheck
}

func validateResponse(initData string) map[string]string {
	if initData == "" {
		return map[string]string{"message": "Сервер работает! Откройте приложение внутри MAX."}
	}

	valid, err := ValidateInitData(initData, os.Getenv("MAX_BOT_TOKEN"), maxInitDataAge())
	switch {
	case err != nil && errors.Is(err, ErrExpiredInitData):
		log.Printf("проверка initData не прошла: %v", err)
		return map[string]string{"message": "Данные устарели. Перезапустите приложение."}
	case err != nil:
		log.Printf("проверка initData не прошла: %v", err)
		return map[string]string{"message": "Данные недействительны"}
	case !valid:
		return map[string]string{"message": "Данные недействительны"}
	}
	return map[string]string{"message": "Данные подтверждены! Пользователь аутентифицирован."}
}

func maxInitDataAge() time.Duration {
	raw := strings.TrimSpace(os.Getenv("MAX_INIT_DATA_MAX_AGE"))
	if raw == "" {
		return time.Hour
	}

	secs, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("MAX_INIT_DATA_MAX_AGE=%q не разобрано, используем 3600", raw)
		return time.Hour
	}
	return time.Duration(secs) * time.Second
}
