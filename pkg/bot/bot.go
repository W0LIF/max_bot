package bot

import (
	"os"

	maxbot "github.com/max-messenger/max-bot-api-client-go"

	"max_bot_api/pkg/storage"
)

// Bot — состояние бота: API-клиент MAX, хранилище и FSM диалогов.
type Bot struct {
	api       *maxbot.Api
	store     storage.Store
	fsm       *FSM
	speechKit *YandexSpeechKit
}

// New создаёт бота. Вызывается из cmd/bot/main.go (long-polling)
// и из api/webhook.go (Vercel).
func New(api *maxbot.Api, store storage.Store) *Bot {
	return &Bot{
		api:   api,
		store: store,
		fsm:   NewFSM(),
		speechKit: NewYandexSpeechKit(
			os.Getenv("YANDEX_SPEECHKIT_API_KEY"),
			os.Getenv("YANDEX_FOLDER_ID"),
		),
	}
}
