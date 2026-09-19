package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"
)

const testToken = "123456:TEST_TOKEN_ABCDEF"

// signInitData собирает initData ровно так, как это делает клиент MAX:
// сортирует параметры по ключу, склеивает "k=v" через \n и подписывает.
func signInitData(t *testing.T, token string, authDate time.Time) string {
	t.Helper()

	fields := map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"chat":      `{"id":12345,"type":"DIALOG"}`,
		"ip":        "192.168.0.1",
		"query_id":  "4c0ab423-342b-4e45-aea4-2747dbc500cd",
		"user":      `{"id":67890,"first_name":"Max","last_name":"User","username":null,"language_code":"ru","photo_url":null}`,
	}

	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	form.Set("hash", signLaunchParams(t, token, form))

	return form.Encode()
}

func signLaunchParams(t *testing.T, token string, form url.Values) string {
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

func TestValidateInitData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	week := 7 * 24 * time.Hour

	tests := []struct {
		name      string
		initData  func(t *testing.T) string
		token     string
		maxAge    time.Duration
		wantValid bool
		wantErrIs error // точный sentinel через errors.Is
		wantErr   bool  // любая ошибка
	}{
		{
			name:      "свежие данные",
			initData:  func(t *testing.T) string { return signInitData(t, testToken, now) },
			token:     testToken,
			maxAge:    time.Hour,
			wantValid: true,
		},
		{
			name:      "на границе окна",
			initData:  func(t *testing.T) string { return signInitData(t, testToken, now.Add(-time.Hour)) },
			token:     testToken,
			maxAge:    time.Hour,
			wantValid: true,
		},
		{
			name: "чуть в будущем — в пределах допуска часов",
			initData: func(t *testing.T) string {
				return signInitData(t, testToken, now.Add(time.Minute))
			},
			token:     testToken,
			maxAge:    time.Hour,
			wantValid: true,
		},
		{
			name:      "устарели",
			initData:  func(t *testing.T) string { return signInitData(t, testToken, now.Add(-2*time.Hour)) },
			token:     testToken,
			maxAge:    time.Hour,
			wantErrIs: ErrExpiredInitData,
		},
		{
			name:      "устарели, но проверку свежести отключили",
			initData:  func(t *testing.T) string { return signInitData(t, testToken, now.Add(-week)) },
			token:     testToken,
			maxAge:    0,
			wantValid: true,
		},
		{
			name: "подделан hash",
			initData: func(t *testing.T) string {
				return replaceHash(t, signInitData(t, testToken, now), func(h string) string {
					return "00" + h[2:]
				})
			},
			token:  testToken,
			maxAge: time.Hour,
		},
		{
			name:     "подписано чужим токеном",
			initData: func(t *testing.T) string { return signInitData(t, "999:OTHER_TOKEN", now) },
			token:    testToken,
			maxAge:   time.Hour,
		},
		{
			name: "подменён user при прежнем hash",
			initData: func(t *testing.T) string {
				return tamperField(t, signInitData(t, testToken, now), "user", `{"id":1}`)
			},
			token:  testToken,
			maxAge: time.Hour,
		},
		{
			name:     "hash не hex",
			initData: func(t *testing.T) string { return "auth_date=1771409719&hash=zzzz" },
			token:    testToken,
			maxAge:   time.Hour,
			wantErr:  true,
		},
		{
			name:     "нет hash",
			initData: func(t *testing.T) string { return "auth_date=1771409719" },
			token:    testToken,
			maxAge:   time.Hour,
			wantErr:  true,
		},
		{
			name:     "пустой initData",
			initData: func(t *testing.T) string { return "" },
			token:    testToken,
			maxAge:   time.Hour,
			wantErr:  true,
		},
		{
			name:      "не задан токен бота",
			initData:  func(t *testing.T) string { return signInitData(t, testToken, now) },
			token:     "",
			maxAge:    time.Hour,
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateInitData(tt.initData(t), tt.token, tt.maxAge, now)

			if valid != tt.wantValid {
				t.Errorf("valid = %v, хотим %v (err = %v)", valid, tt.wantValid, err)
			}

			switch {
			case tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs):
				t.Errorf("err = %v, хотим errors.Is(.., %v)", err, tt.wantErrIs)
			case tt.wantErr && err == nil:
				t.Error("ожидали ошибку, получили nil")
			case !tt.wantErr && tt.wantErrIs == nil && err != nil:
				t.Errorf("не ожидали ошибку, получили %v", err)
			}
		})
	}
}

// Проверяем, что дубликат параметра отклоняется: канонизация становится
// неоднозначной, а значит подпись перестаёт быть привязана к данным.
func TestValidateInitData_DuplicateParam(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	data := signInitData(t, testToken, now)
	// Тот же auth_date ещё раз, с другим значением.
	data += "&auth_date=" + strconv.FormatInt(now.Add(time.Second).Unix(), 10)
	// Пересчитываем hash от набора с дубликатом, чтобы пройти подпись,
	// если бы проверка дубликатов отсутствовала.
	form, err := url.ParseQuery(data)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	data = replaceHash(t, data, func(h string) string { return signLaunchParams(t, testToken, form) })

	if _, err := validateInitData(data, testToken, time.Hour, now); err == nil {
		t.Error("ожидали ошибку на дубликате параметра, получили nil")
	}
}

func TestValidateInitData_MissingAuthDate(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	form := url.Values{}
	form.Set("user", `{"id":1}`)
	form.Set("hash", signLaunchParams(t, testToken, form))

	if _, err := validateInitData(form.Encode(), testToken, time.Hour, now); err == nil {
		t.Error("ожидали ошибку на отсутствующем auth_date, получили nil")
	}
}

// replaceHash меняет значение hash в urlencoded-строке.
func replaceHash(t *testing.T, initData string, fn func(string) string) string {
	t.Helper()

	form, err := url.ParseQuery(initData)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	form.Set("hash", fn(form.Get("hash")))

	return form.Encode()
}

// tamperField меняет значение поля, оставляя hash от прежней подписи.
// Так выглядит правка данных после того, как MAX их подписал.
func tamperField(t *testing.T, initData, key, value string) string {
	t.Helper()

	form, err := url.ParseQuery(initData)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}

	original := form.Get("hash")
	form.Set(key, value)
	form.Set("hash", original)

	return form.Encode()
}
