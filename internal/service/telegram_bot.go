package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TelegramBotService struct {
	token  string
	client *http.Client
}

func NewTelegramBotService(token string) *TelegramBotService {
	return &TelegramBotService{
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *TelegramBotService) SendTelegramMessage(ctx context.Context, telegramID int64, text string) error {
	if s.token == "" {
		return fmt.Errorf("telegram bot token is not configured")
	}
	if telegramID <= 0 {
		return fmt.Errorf("invalid telegram id")
	}
	if text == "" {
		return fmt.Errorf("message text is empty")
	}

	payload := map[string]any{
		"chat_id": telegramID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("telegram sendMessage failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode telegram response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("telegram sendMessage returned ok=false")
	}

	return nil
}
