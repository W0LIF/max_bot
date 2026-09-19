package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrExpiredInitData — подпись верна, но данные слишком старые.
// Позволяет отличить подделку от протухшего, но подлинного initData.
var ErrExpiredInitData = errors.New("initData устарел")

// clockSkewTolerance — допуск на расхождение часов клиента и сервера.
const clockSkewTolerance = 2 * time.Minute

// ValidateInitData проверяет подпись initData мини-приложения MAX.
//
// Схема подписи (https://dev.max.ru/docs/webapps/validation):
//
//	secretKey    = HMAC_SHA256(key="WebAppData", data=botToken)
//	signature    = HMAC_SHA256(key=secretKey, data=launchParams)
//	launchParams = отсортированные по ключу "k=v", склеенные "\n", без hash
//
// maxAge задаёт окно жизни данных по auth_date; <= 0 отключает проверку.
// Возвращает ErrExpiredInitData (через errors.Is), если подпись подлинная,
// но данные старше окна.
func ValidateInitData(initData, botToken string, maxAge time.Duration) (bool, error) {
	return validateInitData(initData, botToken, maxAge, time.Now())
}

func validateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (bool, error) {
	if botToken == "" {
		return false, errors.New("MAX_BOT_TOKEN не задан")
	}

	params, err := url.ParseQuery(initData)
	if err != nil {
		return false, fmt.Errorf("не удалось разобрать initData: %w", err)
	}

	// По спецификации каждый параметр встречается ровно один раз.
	// Дубликаты делают канонизацию неоднозначной — отклоняем.
	for k, v := range params {
		if len(v) != 1 {
			return false, fmt.Errorf("параметр %q встречается %d раз", k, len(v))
		}
	}

	originalHash := params.Get("hash")
	if originalHash == "" {
		return false, errors.New("hash не найден")
	}
	params.Del("hash")

	rawHash, err := hex.DecodeString(originalHash)
	if err != nil {
		return false, fmt.Errorf("hash не является hex: %w", err)
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var launchParams strings.Builder
	for i, k := range keys {
		if i > 0 {
			launchParams.WriteString("\n")
		}
		launchParams.WriteString(k + "=" + params.Get(k))
	}

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	signer := hmac.New(sha256.New, secretKey)
	signer.Write([]byte(launchParams.String()))
	signature := signer.Sum(nil)

	// Constant-time сравнение, чтобы не светить тайминг-атаки.
	if !hmac.Equal(signature, rawHash) {
		return false, nil
	}

	// Проверка свежести — только для подлинных данных.
	if maxAge > 0 {
		authDate, err := parseAuthDate(params)
		if err != nil {
			return false, err
		}

		age := now.Sub(authDate)
		if age > maxAge+clockSkewTolerance || age < -clockSkewTolerance {
			return false, fmt.Errorf("%w: возраст %s", ErrExpiredInitData, age.Round(time.Second))
		}
	}

	return true, nil
}

func parseAuthDate(params url.Values) (time.Time, error) {
	raw := params.Get("auth_date")
	if raw == "" {
		return time.Time{}, errors.New("auth_date не найден")
	}

	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("auth_date не является unix-временем: %w", err)
	}

	return time.Unix(secs, 0), nil
}
