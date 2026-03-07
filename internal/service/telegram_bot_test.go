package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramBotServiceFindGroupChatByTitleAmbiguous(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"result": [
				{"message": {"chat": {"id": -1001, "type": "group", "title": "GridLogger"}}},
				{"my_chat_member": {"chat": {"id": -1002, "type": "supergroup", "title": "GridLogger"}}}
			]
		}`))
	}))
	defer server.Close()

	bot := NewTelegramBotService("token")
	bot.client = server.Client()
	bot.baseURL = server.URL

	_, err := bot.FindGroupChatByTitle(context.Background(), "GridLogger")
	if !errors.Is(err, ErrTelegramBotGroupAmbiguous) {
		t.Fatalf("expected ErrTelegramBotGroupAmbiguous, got %v", err)
	}
}

func TestTelegramBotServiceFindGroupChatByTitleNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true, "result": []}`))
	}))
	defer server.Close()

	bot := NewTelegramBotService("token")
	bot.client = server.Client()
	bot.baseURL = server.URL

	_, err := bot.FindGroupChatByTitle(context.Background(), "Missing Group")
	if !errors.Is(err, ErrTelegramBotGroupNotFound) {
		t.Fatalf("expected ErrTelegramBotGroupNotFound, got %v", err)
	}
}
