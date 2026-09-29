// Package bot содержит общую логику бота MAX: обработку апдейтов,
// ответы на сообщения и проверку подписи мини-приложения.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	log.Printf("DEBUG dispatch: %T", update)

	switch u := update.(type) {
	case *schemes.MessageCreatedUpdate:
		b.onMessage(ctx, u)
	case *schemes.MessageCallbackUpdate:
		log.Printf("DEBUG callback payload=%q", u.Callback.Payload)
		b.onCallback(ctx, u)
	default:
		log.Printf("DEBUG unknown update: %+v", update)
	}
}

// --- Обработка текстовых сообщений ---

func (b *Bot) onMessage(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	userID := upd.Message.Sender.UserId
	user, isNew, err := loadOrCreateBotUser(ctx, b.store, userID)
	if err != nil {
		log.Printf("load user %d: %v", userID, err)
		b.reply(ctx, upd, "Не удалось открыть профиль. Попробуй ещё раз позже.", nil)
		return
	}
	if isNew || !user.Consent {
		b.reply(ctx, upd, consentPrompt(isNew), kbConsent())
		return
	}

	text := strings.TrimSpace(upd.Message.Body.Text)
	if text == "" {
		if audioURL := audioAttachmentURL(upd.Message.Body.Attachments); audioURL != "" {
			if b.speechKit == nil {
				b.sendToUser(ctx, userID, "Распознавание голоса пока не настроено.", nil)
				return
			}

			transcript, err := b.speechKit.TranscribeURL(ctx, audioURL)
			if err != nil {
				log.Printf("Yandex SpeechKit recognition error: %v", err)
				b.sendToUser(ctx, userID, "Не получилось распознать голосовое сообщение. Попробуй ещё раз или отправь текст.", nil)
				return
			}
			text = strings.TrimSpace(transcript)
			if text == "" {
				b.sendToUser(ctx, userID, "В голосовом сообщении не удалось распознать речь.", nil)
				return
			}
		}
	}
	lowerText := strings.ToLower(text)

	if text == "/start" || strings.EqualFold(text, "start") {
		b.handleStart(ctx, upd)
		return
	}

	// Обработка "привет" — отвечаем приветствием и показываем картинку
	if lowerText == "привет" {
		b.handleHello(ctx, upd)
		return
	}

	// Обработка "меню" — показываем главное меню
	if lowerText == "меню" {
		b.reply(ctx, upd, "Главное меню:", kbMainMenu())
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
	b.reply(ctx, upd, "С возвращением! Что делаем?", kbMainMenu())
}

func loadOrCreateBotUser(ctx context.Context, store storage.Store, userID int64) (*storage.User, bool, error) {
	user, err := store.GetUser(ctx, userID)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, storage.ErrUserNotFound) {
		return nil, false, err
	}

	user = &storage.User{ID: userID, OnboardedAt: time.Now()}
	if err := store.SaveUser(ctx, user); err != nil {
		return nil, false, err
	}
	return user, true, nil
}

func consentPrompt(isNew bool) string {
	if isNew {
		return "Привет! Я помогу следить за дедлайнами и настроением. Согласны на обработку данных? Голосовые сообщения будут передаваться в Yandex SpeechKit для распознавания."
	}
	return "Чтобы пользоваться задачами и напоминаниями, нужно согласие на обработку данных. Голосовые сообщения будут передаваться в Yandex SpeechKit для распознавания."
}

func (b *Bot) fallback(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	b.reply(ctx, upd, "Не понял. Вот меню:", kbMainMenu())
}

// handleHello — обработка приветствия с отправкой картинки.
func (b *Bot) handleHello(ctx context.Context, upd *schemes.MessageCreatedUpdate) {
	userID := upd.Message.Sender.UserId

	// Сначала отправляем текстовое приветствие
	b.sendToUser(ctx, userID, "Привет! 👋", nil)

	// Затем отправляем картинку
	// Используем hero.png из frontend assets
	imagePath := filepath.Join("frontend", "src", "assets", "hero.png")
	if err := b.sendPhotoFromFile(ctx, userID, imagePath); err != nil {
		log.Printf("send photo error: %v", err)
		// Если не получилось отправить картинку, просто показываем меню
		b.sendToUser(ctx, userID, "Вот главное меню:", kbMainMenu())
		return
	}

	// После картинки показываем меню
	b.sendToUser(ctx, userID, "Что делаем?", kbMainMenu())
}

// --- Обработка нажатий на кнопки ---

func (b *Bot) onCallback(ctx context.Context, upd *schemes.MessageCallbackUpdate) {
	_, err := b.api.Messages.AnswerOnCallback(
		ctx,
		upd.Callback.CallbackID,
		&schemes.CallbackAnswer{
			Notification: "Принято",
		},
	)
	if err != nil {
		log.Printf("AnswerOnCallback error: %v", err)
	}

	userID := upd.Callback.User.UserId
	payload := upd.Callback.Payload
	user, isNew, userErr := loadOrCreateBotUser(ctx, b.store, userID)
	if userErr != nil {
		log.Printf("load user %d: %v", userID, userErr)
		b.sendToUser(ctx, userID, "Не удалось открыть профиль. Попробуй ещё раз позже.", nil)
		return
	}
	if (isNew || !user.Consent) && payload != CbConsentYes && payload != CbConsentNo {
		b.sendToUser(ctx, userID, consentPrompt(isNew), kbConsent())
		return
	}

	switch payload {
	case CbConsentYes:
		if err := b.store.SetConsent(ctx, userID, true); err != nil {
			log.Printf("SetConsent: %v", err)
			b.sendToUser(ctx, userID, "Не удалось сохранить согласие. Попробуй ещё раз.", kbConsent())
			return
		}
		b.sendToUser(ctx, userID,
			"Отлично! Присылать напоминания о дедлайнах?", kbReminders())

	case CbConsentNo:
		if err := b.store.SetConsent(ctx, userID, false); err != nil {
			log.Printf("SetConsent: %v", err)
			b.sendToUser(ctx, userID, "Не удалось сохранить выбор. Попробуй ещё раз.", kbConsent())
			return
		}
		if err := b.store.SetReminders(ctx, userID, false); err != nil {
			log.Printf("disable reminders for %d: %v", userID, err)
		}
		if err := b.store.LeaveGroups(ctx, userID); err != nil {
			log.Printf("remove user %d from groups: %v", userID, err)
		}
		b.sendToUser(ctx, userID, "Понял. Чтобы запомнить выбор, я сохраню технический ID MAX и факт отказа. Без согласия задачи, настроение и голосовые сообщения обрабатываться не будут. Если передумаешь, отправь любое сообщение и выбери «Да».", nil)

	case CbRemindYes:
		if err := b.store.SetReminders(ctx, userID, true); err != nil {
			log.Printf("SetReminders: %v", err)
			b.sendToUser(ctx, userID, "Не удалось сохранить настройку. Попробуй ещё раз.", kbReminders())
			return
		}
		b.sendToUser(ctx, userID, "Готово! Буду напоминать о дедлайнах.", kbMainMenu())

	case CbRemindNo:
		if err := b.store.SetReminders(ctx, userID, false); err != nil {
			log.Printf("SetReminders: %v", err)
			b.sendToUser(ctx, userID, "Не удалось сохранить настройку. Попробуй ещё раз.", kbReminders())
			return
		}
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
	case CbMenuMood:
		b.sendToUser(ctx, userID, "Как ты себя чувствуешь?", kbMood())

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

// sendPhotoFromFile загружает и отправляет фото из файла.
func (b *Bot) sendPhotoFromFile(ctx context.Context, userID int64, filePath string) error {
	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	// Загружаем фото на сервер MAX
	photoTokens, err := b.api.Uploads.UploadPhotoFromFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("upload photo: %w", err)
	}

	// Получаем первый токен фото
	var token string
	for _, pt := range photoTokens.Photos {
		token = pt.Token
		break
	}
	if token == "" {
		return fmt.Errorf("no photo token received")
	}

	// Отправляем сообщение с фото
	msg := maxbot.NewMessage().SetUser(userID).AddPhotoByToken(token)
	if err := b.api.Messages.Send(ctx, msg); err != nil {
		return fmt.Errorf("send photo message: %w", err)
	}

	return nil
}

// --- Webhook ---

// HandleWebhook — конструктор обработчика webhook от MAX
// (маршрут /api/webhook).
//
// Стор передаётся снаружи — чтобы состояние пользователя сохранялось
// между запросами. Раньше здесь создавался NewMemoryStore() на каждый
// запрос, и онбординг терялся (B0 в Этапе 2).
//
// Использование:
//
//	mux.HandleFunc("POST /api/webhook", bot.HandleWebhook(store))
func HandleWebhook(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		b := New(api, store)

		secret := os.Getenv("MAX_BOT_API_SECRET")

		api.GetUpdateHandlerFunc(func(update schemes.UpdateInterface) {
			b.Dispatch(r.Context(), update)
			w.WriteHeader(http.StatusOK)
		}, secret)(w, r)
	}
}

// --- Проверка initData мини-приложения (legacy-маршрут /api/validate) ---

// HandleValidate — старый отладочный эндпоинт. Оставлен для обратной
// совместимости и для App.tsx, пока фронт не перейдёт на /api/auth/validate.
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
