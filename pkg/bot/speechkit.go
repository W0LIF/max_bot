package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

const (
	yandexSpeechKitEndpoint = "https://stt.api.cloud.yandex.net/speech/v1/stt:recognize"
	maxAudioSize            = 10 << 20
)

type YandexSpeechKit struct {
	apiKey   string
	folderID string
	endpoint string
	client   *http.Client
}

func NewYandexSpeechKit(apiKey, folderID string) *YandexSpeechKit {
	apiKey = strings.TrimSpace(apiKey)
	folderID = strings.TrimSpace(folderID)
	if apiKey == "" || folderID == "" {
		return nil
	}

	return &YandexSpeechKit{
		apiKey:   apiKey,
		folderID: folderID,
		endpoint: yandexSpeechKitEndpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *YandexSpeechKit) TranscribeURL(ctx context.Context, audioURL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(audioURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" || parsedURL.User != nil {
		return "", errors.New("invalid audio URL")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create audio download request: %w", err)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download audio: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download audio: HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxAudioSize {
		return "", errors.New("audio exceeds size limit")
	}

	audio, err := io.ReadAll(io.LimitReader(response.Body, maxAudioSize+1))
	if err != nil {
		return "", fmt.Errorf("read audio: %w", err)
	}
	if len(audio) == 0 {
		return "", errors.New("audio is empty")
	}
	if len(audio) > maxAudioSize {
		return "", errors.New("audio exceeds size limit")
	}

	return s.recognize(ctx, audio)
}

func (s *YandexSpeechKit) recognize(ctx context.Context, audio []byte) (string, error) {
	endpoint, err := url.Parse(s.endpoint)
	if err != nil {
		return "", fmt.Errorf("parse SpeechKit endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("folderId", s.folderID)
	query.Set("lang", "ru-RU")
	query.Set("format", "oggopus")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("create SpeechKit request: %w", err)
	}
	request.Header.Set("Authorization", "Api-Key "+s.apiKey)
	request.Header.Set("Content-Type", "application/octet-stream")

	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("call SpeechKit: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SpeechKit returned HTTP %d", response.StatusCode)
	}

	var result struct {
		Text string `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode SpeechKit response: %w", err)
	}
	return result.Text, nil
}

func audioAttachmentURL(attachments []interface{}) string {
	for _, attachment := range attachments {
		switch audio := attachment.(type) {
		case *schemes.AudioAttachment:
			return audio.Payload.Url
		case schemes.AudioAttachment:
			return audio.Payload.Url
		}
	}
	return ""
}
