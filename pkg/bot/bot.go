package bot

import (
	maxbot "github.com/max-messenger/max-bot-api-client-go"

	"max_bot_api/pkg/storage"
)

// Bot — состояние бота: API-клиент MAX, хранилище и FSM диалогов.
type Bot struct {
	api   *maxbot.Api
	store storage.Store
	fsm   *FSM
}

// New создаёт бота. Вызывается из cmd/bot/main.go (long-polling)
// и из api/webhook.go (Vercel).
func New(api *maxbot.Api, store storage.Store) *Bot {
	return &Bot{
		api:   api,
		store: store,
		fsm:   NewFSM(),
	}
}
