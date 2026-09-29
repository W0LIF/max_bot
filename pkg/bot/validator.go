package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// DefaultUserName — подставляется, если в initData нет first_name.
const DefaultUserName = "Студент"

// User — минимальный профиль пользователя, извлечённый из initData.
// Используется middleware httpapi и GET /api/me, чтобы не парсить initData
// дважды.
type User struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
}

// --- Публичный API ---

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
//
// Если нужен и профиль пользователя — используй ValidateAndParseInitData,
// чтобы не разбирать initData дважды.
func ValidateInitData(initData, botToken string, maxAge time.Duration) (bool, error) {
	_, err := ValidateAndParseInitData(initData, botToken, maxAge, time.Now())
	if err != nil {
		// ErrExpiredInitData и прочие sentinel-ошибки должны долетать
		// до вызывающего как есть.
		return false, err
	}
	return true, nil
}

// ValidateAndParseInitData проверяет подпись initData и, если она валидна,
// извлекает профиль пользователя. Один разбор urlencoded-строки вместо двух.
//
// now передаётся явно, чтобы тесты были детерминированными; в проде
// используй time.Now().
//
// Возвращаемые ошибки:
//   - ErrExpiredInitData — подпись верна, но auth_date вне окна maxAge;
//   - прочие ошибки — невалидная подпись, отсутствующие/дублирующиеся
//     параметры, битый JSON в user, отсутствующий user.id.
func ValidateAndParseInitData(initData, botToken string, maxAge time.Duration, now time.Time) (User, error) {
	if botToken == "" {
		return User{}, errors.New("MAX_BOT_TOKEN не задан")
	}

	params, err := parseLaunchParams(initData)
	if err != nil {
		return User{}, err
	}

	if err := verifySignature(params, botToken); err != nil {
		return User{}, err
	}

	if maxAge > 0 {
		if err := checkAuthDate(params, maxAge, now); err != nil {
			return User{}, err
		}
	}

	return extractUser(params)
}

// ParseInitData разбирает initData и достаёт user.id / user.first_name.
//
// ВНИМАНИЕ: подпись НЕ проверяется. Использовать только там, где initData
// уже прошёл ValidateInitData / ValidateAndParseInitData (например, при
// повторном чтении из доверенного источника). В HTTP-middleware вызывай
// ValidateAndParseInitData.
func ParseInitData(raw string) (User, error) {
	params, err := parseLaunchParams(raw)
	if err != nil {
		return User{}, err
	}
	return extractUser(params)
}

// --- Внутренние шаги ---

// parseLaunchParams разбирает urlencoded-строку initData и проверяет, что
// каждый параметр встречается ровно один раз. Единая точка разбора формата
// для всех остальных функций пакета.
func parseLaunchParams(raw string) (url.Values, error) {
	params, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать initData: %w", err)
	}

	// По спецификации каждый параметр встречается ровно один раз.
	// Дубликаты делают канонизацию неоднозначной — отклоняем.
	for k, v := range params {
		if len(v) != 1 {
			return nil, fmt.Errorf("параметр %q встречается %d раз", k, len(v))
		}
	}

	return params, nil
}

// verifySignature проверяет HMAC-подпись launchParams. Мутирует params:
// удаляет hash перед канонизацией. Возвращает nil, если подпись совпала,
// и ошибку во всех остальных случаях (в т.ч. при невалидной подписи —
// ошибка не sentinel, а обычная).
func verifySignature(params url.Values, botToken string) error {
	originalHash := params.Get("hash")
	if originalHash == "" {
		return errors.New("hash не найден")
	}

	rawHash, err := hex.DecodeString(originalHash)
	if err != nil {
		return fmt.Errorf("hash не является hex: %w", err)
	}

	params.Del("hash")
	launch := canonicalLaunchParams(params)

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	signer := hmac.New(sha256.New, secretKey)
	signer.Write([]byte(launch))
	signature := signer.Sum(nil)

	// Constant-time сравнение, чтобы не светить тайминг-атаки.
	if !hmac.Equal(signature, rawHash) {
		return errors.New("подпись initData недействительна")
	}
	return nil
}

// checkAuthDate проверяет свежесть данных. maxAge > 0 уже гарантирован
// вызывающим.
func checkAuthDate(params url.Values, maxAge time.Duration, now time.Time) error {
	authDate, err := parseAuthDate(params)
	if err != nil {
		return err
	}

	age := now.Sub(authDate)
	if age > maxAge+clockSkewTolerance || age < -clockSkewTolerance {
		return fmt.Errorf("%w: возраст %s", ErrExpiredInitData, age.Round(time.Second))
	}
	return nil
}

// extractUser достаёт профиль из уже проверенных params.
func extractUser(params url.Values) (User, error) {
	rawUser := params.Get("user")
	if rawUser == "" {
		return User{}, errors.New("user не найден в initData")
	}

	var u struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	}
	if err := json.Unmarshal([]byte(rawUser), &u); err != nil {
		return User{}, fmt.Errorf("user не является JSON: %w", err)
	}
	if u.ID == 0 {
		return User{}, errors.New("user.id отсутствует")
	}
	if u.FirstName == "" {
		u.FirstName = DefaultUserName
	}

	return User{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Username:  u.Username,
	}, nil
}

// canonicalLaunchParams собирает launchParams: отсортированные по ключу
// "k=v", склеенные "\n". Значения берутся как есть (url.Values.Get уже
// раскодировал percent-encoding).
func canonicalLaunchParams(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(k + "=" + params.Get(k))
	}
	return sb.String()
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

func validateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (bool, error) {
	_, err := ValidateAndParseInitData(initData, botToken, maxAge, now)
	if err != nil {
		return false, err
	}
	return true, nil
}
