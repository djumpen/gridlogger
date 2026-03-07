package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TelegramBotService struct {
	token   string
	client  *http.Client
	baseURL string
}

type TelegramGroupChat struct {
	TelegramID int64
	Title      string
	ChatType   string
	Username   string
}

func NewTelegramBotService(token string) *TelegramBotService {
	return &TelegramBotService{
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

func (s *TelegramBotService) Enabled() bool {
	return s != nil && strings.TrimSpace(s.token) != ""
}

func (s *TelegramBotService) SendTelegramMessage(ctx context.Context, telegramID int64, text string) error {
	if !s.Enabled() {
		return fmt.Errorf("telegram bot token is not configured")
	}
	if telegramID == 0 {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.methodURL("sendMessage"), bytes.NewReader(body))
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

func (s *TelegramBotService) FindGroupChatByTitle(ctx context.Context, title string) (TelegramGroupChat, error) {
	if !s.Enabled() {
		return TelegramGroupChat{}, fmt.Errorf("telegram bot token is not configured")
	}

	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return TelegramGroupChat{}, fmt.Errorf("group title is required")
	}

	reqURL, err := url.Parse(s.methodURL("getUpdates"))
	if err != nil {
		return TelegramGroupChat{}, err
	}
	query := reqURL.Query()
	query.Set("limit", "100")
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return TelegramGroupChat{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return TelegramGroupChat{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TelegramGroupChat{}, fmt.Errorf("telegram getUpdates failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var out struct {
		OK          bool             `json:"ok"`
		Description string           `json:"description"`
		Result      []telegramUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TelegramGroupChat{}, fmt.Errorf("decode telegram getUpdates response: %w", err)
	}
	if !out.OK {
		if out.Description != "" {
			return TelegramGroupChat{}, fmt.Errorf("telegram getUpdates returned ok=false: %s", out.Description)
		}
		return TelegramGroupChat{}, fmt.Errorf("telegram getUpdates returned ok=false")
	}

	seen := map[int64]TelegramGroupChat{}
	for _, update := range out.Result {
		for _, chat := range update.groupChats() {
			if !matchesTelegramGroupTitle(normalizedTitle, chat.Title) {
				continue
			}
			seen[chat.TelegramID] = chat
		}
	}

	if len(seen) == 0 {
		return TelegramGroupChat{}, ErrTelegramBotGroupNotFound
	}
	if len(seen) > 1 {
		return TelegramGroupChat{}, ErrTelegramBotGroupAmbiguous
	}
	for _, chat := range seen {
		return chat, nil
	}
	return TelegramGroupChat{}, ErrTelegramBotGroupNotFound
}

func (s *TelegramBotService) methodURL(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", strings.TrimRight(s.baseURL, "/"), s.token, method)
}

func matchesTelegramGroupTitle(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

type telegramUpdate struct {
	Message           *telegramMessage           `json:"message"`
	EditedMessage     *telegramMessage           `json:"edited_message"`
	ChannelPost       *telegramMessage           `json:"channel_post"`
	EditedChannelPost *telegramMessage           `json:"edited_channel_post"`
	MyChatMember      *telegramChatMemberUpdated `json:"my_chat_member"`
	ChatMember        *telegramChatMemberUpdated `json:"chat_member"`
	CallbackQuery     *telegramCallbackQuery     `json:"callback_query"`
}

func (u telegramUpdate) groupChats() []TelegramGroupChat {
	chats := make([]TelegramGroupChat, 0, 7)
	add := func(chat *telegramChat) {
		if chat == nil {
			return
		}
		if chat.Type != "group" && chat.Type != "supergroup" {
			return
		}
		chats = append(chats, TelegramGroupChat{
			TelegramID: chat.ID,
			Title:      strings.TrimSpace(chat.Title),
			ChatType:   strings.TrimSpace(chat.Type),
			Username:   strings.TrimSpace(chat.Username),
		})
	}

	if u.Message != nil {
		add(&u.Message.Chat)
	}
	if u.EditedMessage != nil {
		add(&u.EditedMessage.Chat)
	}
	if u.ChannelPost != nil {
		add(&u.ChannelPost.Chat)
	}
	if u.EditedChannelPost != nil {
		add(&u.EditedChannelPost.Chat)
	}
	if u.MyChatMember != nil {
		add(&u.MyChatMember.Chat)
	}
	if u.ChatMember != nil {
		add(&u.ChatMember.Chat)
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil {
		add(&u.CallbackQuery.Message.Chat)
	}
	return chats
}

type telegramMessage struct {
	Chat telegramChat `json:"chat"`
}

type telegramChatMemberUpdated struct {
	Chat telegramChat `json:"chat"`
}

type telegramCallbackQuery struct {
	Message *telegramMessage `json:"message"`
}

type telegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}
