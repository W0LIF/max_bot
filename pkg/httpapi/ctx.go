package httpapi

import (
	"context"

	"max_bot_api/pkg/bot"
)

// ctxKey — приватный тип для ключей context, чтобы не пересечься
// с ключами из других пакетов.
type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeyUser
)

// userIDFrom достаёт user_id, положенный middleware RequireInitData.
// Возвращает 0, если middleware не сработал (маршрут не защищён —
// такого быть не должно).
func userIDFrom(ctx context.Context) int64 {
	id, _ := ctx.Value(ctxKeyUserID).(int64)
	return id
}

// userFrom достаёт профиль пользователя из initData.
func userFrom(ctx context.Context) bot.User {
	u, _ := ctx.Value(ctxKeyUser).(bot.User)
	return u
}
