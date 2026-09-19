package bot

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"errors"
	"net/http"
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

// russianRootCA — корневой сертификат Минцифры (Russian Trusted Root CA).
// API MAX (platform-api2.max.ru) использует цепочку от этого УЦ, которого нет
// в системном хранилище доверенных корней вне России (в т.ч. на Vercel).
// Источник: цепочка сертификата *.max.ru.
//
//go:embed certs/russian_trusted_root_ca.pem
var russianRootCA []byte

// NewAPIClient создаёт клиент Bot API с системными корнями + Russian Trusted
// Root CA. Без этого на зарубежных хостингах отправка сообщений падает с
// "tls: failed to verify certificate: x509: certificate signed by unknown
// authority".
func NewAPIClient(token string) (*maxbot.Api, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(russianRootCA) {
		return nil, errors.New("не удалось добавить Russian Trusted Root CA в пул доверенных")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	return maxbot.New(token, maxbot.WithHTTPClient(httpClient))
}
