package storage

import (
	"time"
)

type User struct {
	ID          int64     `json:"id"`           // ID пользователя MAX
	Consent     bool      `json:"consent"`      // согласие на обработку данных
	RemindersOn bool      `json:"reminders_on"` // напоминания о дедлайнах
	OnboardedAt time.Time `json:"onboarded_at"` // дата онбординга

}

// Проверяет, что OnboardedAt не нулевое
func (u *User) IsOnboarded() bool {
	return !u.OnboardedAt.IsZero()
}
