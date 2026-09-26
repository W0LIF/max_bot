package bot

import (
	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

// Bot — состояние бота: API-клиент MAX, хранилище и FSM диалогов.
//
// Пока Разр1 не сделал pkg/storage, поле store имеет локальный тип Store
// из store_local.go. Когда появится настоящий storage.Store — меняем
// тип поля на storage.Store, остальная логика не трогается.
type Bot struct {
	api   *maxbot.Api
	store Store
	fsm   *FSM
}

// New создаёт бота. Вызывается из cmd/bot/main.go (long-polling)
// и из api/webhook.go (Vercel).
func New(api *maxbot.Api, store Store) *Bot {
	return &Bot{
		api:   api,
		store: store,
		fsm:   NewFSM(),
	}
}
