package bot

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func TestYandexSpeechKitTranscribeURL(t *testing.T) {
	audio := []byte("ogg opus audio")
	mediaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/voice.ogg" {
			t.Errorf("media request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write(audio)
	}))
	defer mediaServer.Close()

	speechServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.Method != http.MethodPost || r.URL.Path != "/recognize" {
			t.Errorf("SpeechKit request = %s %s", r.Method, r.URL.Path)
		}
		if query.Get("folderId") != "folder-123" || query.Get("lang") != "ru-RU" || query.Get("format") != "oggopus" {
			t.Errorf("SpeechKit query = %v", query)
		}
		if got := r.Header.Get("Authorization"); got != "Api-Key test-api-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("Content-Type = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read SpeechKit body: %v", err)
		}
		if !bytes.Equal(body, audio) {
			t.Errorf("SpeechKit audio body = %q, want %q", body, audio)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"Создать задачу завтра"}`))
	}))
	defer speechServer.Close()

	speechKit := NewYandexSpeechKit("test-api-key", "folder-123")
	speechKit.endpoint = speechServer.URL + "/recognize"
	got, err := speechKit.TranscribeURL(context.Background(), mediaServer.URL+"/voice.ogg")
	if err != nil {
		t.Fatalf("TranscribeURL: %v", err)
	}
	if got != "Создать задачу завтра" {
		t.Fatalf("transcript = %q", got)
	}
}

func TestNewYandexSpeechKitRequiresCredentials(t *testing.T) {
	if NewYandexSpeechKit("", "folder-123") != nil {
		t.Fatal("expected nil without API key")
	}
	if NewYandexSpeechKit("test-api-key", "") != nil {
		t.Fatal("expected nil without folder ID")
	}
}

func TestAudioAttachmentURL(t *testing.T) {
	attachments := []interface{}{
		&schemes.FileAttachment{},
		&schemes.AudioAttachment{Payload: schemes.MediaAttachmentPayload{Url: "https://media.example/voice.ogg"}},
	}
	if got := audioAttachmentURL(attachments); got != "https://media.example/voice.ogg" {
		t.Fatalf("audioAttachmentURL = %q", got)
	}
	if got := audioAttachmentURL(nil); got != "" {
		t.Fatalf("audioAttachmentURL(nil) = %q", got)
	}
	if _, err := url.ParseRequestURI(audioAttachmentURL(attachments)); err != nil {
		t.Fatalf("audio URL is invalid: %v", err)
	}
}
