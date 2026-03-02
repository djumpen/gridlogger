package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrTelegramAuthDisabled = errors.New("telegram auth is disabled")
	ErrTelegramInvalidData  = errors.New("invalid telegram auth payload")
	ErrTelegramHashMismatch = errors.New("telegram auth hash mismatch")
	ErrTelegramAuthStale    = errors.New("telegram auth data is stale")
	ErrTelegramAuthFuture   = errors.New("telegram auth_date is in the future")
	ErrTelegramReplay       = errors.New("telegram auth replay detected")
	ErrTelegramUserBlocked  = errors.New("telegram account is blocked")
)

type TelegramAccount struct {
	UserID       int64     `json:"id"`
	TelegramID   int64     `json:"telegramId"`
	Username     string    `json:"username"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	PhotoURL     string    `json:"photoUrl"`
	IsBlocked    bool      `json:"isBlocked"`
	IsAdmin      bool      `json:"isAdmin"`
	LastAuthDate int64     `json:"lastAuthDate"`
	LastLoginAt  time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type TelegramAccountUpsert struct {
	TelegramID   int64
	Username     string
	FirstName    string
	LastName     string
	PhotoURL     string
	LastAuthDate int64
	LastLoginAt  time.Time
}

type TelegramAccountStore interface {
	UpsertTelegramAccount(ctx context.Context, in TelegramAccountUpsert) (TelegramAccount, bool, error)
	GetTelegramAccountByUserID(ctx context.Context, userID int64) (TelegramAccount, bool, error)
}

type TelegramAuthService struct {
	store    TelegramAccountStore
	botToken string
	authTTL  time.Duration
	nowFn    func() time.Time
}

func NewTelegramAuthService(store TelegramAccountStore, botToken string, authTTL time.Duration) *TelegramAuthService {
	return &TelegramAuthService{
		store:    store,
		botToken: botToken,
		authTTL:  authTTL,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *TelegramAuthService) Enabled() bool {
	return s.botToken != "" && s.store != nil
}

func (s *TelegramAuthService) Authenticate(ctx context.Context, fields map[string]string) (TelegramAccount, error) {
	if !s.Enabled() {
		return TelegramAccount{}, ErrTelegramAuthDisabled
	}

	in, err := parseTelegramFields(fields)
	if err != nil {
		return TelegramAccount{}, err
	}

	if !verifyTelegramHash(fields, s.botToken) {
		return TelegramAccount{}, ErrTelegramHashMismatch
	}

	now := s.nowFn().UTC()
	authAt := time.Unix(in.LastAuthDate, 0).UTC()
	if authAt.After(now.Add(60 * time.Second)) {
		return TelegramAccount{}, ErrTelegramAuthFuture
	}
	if now.Sub(authAt) > s.authTTL {
		return TelegramAccount{}, ErrTelegramAuthStale
	}

	account, replay, err := s.store.UpsertTelegramAccount(ctx, in)
	if err != nil {
		return TelegramAccount{}, fmt.Errorf("upsert telegram account: %w", err)
	}
	if replay {
		return TelegramAccount{}, ErrTelegramReplay
	}
	if account.IsBlocked {
		return TelegramAccount{}, ErrTelegramUserBlocked
	}

	return account, nil
}

func (s *TelegramAuthService) GetAccountByUserID(ctx context.Context, userID int64) (TelegramAccount, bool, error) {
	if !s.Enabled() {
		return TelegramAccount{}, false, ErrTelegramAuthDisabled
	}
	if userID <= 0 {
		return TelegramAccount{}, false, ErrTelegramInvalidData
	}
	account, found, err := s.store.GetTelegramAccountByUserID(ctx, userID)
	if err != nil {
		return TelegramAccount{}, false, fmt.Errorf("get telegram account by user id: %w", err)
	}
	return account, found, nil
}

func parseTelegramFields(fields map[string]string) (TelegramAccountUpsert, error) {
	if len(fields) == 0 {
		return TelegramAccountUpsert{}, ErrTelegramInvalidData
	}

	rawID := strings.TrimSpace(fields["id"])
	rawAuthDate := strings.TrimSpace(fields["auth_date"])
	rawHash := strings.TrimSpace(fields["hash"])
	if rawID == "" || rawAuthDate == "" || rawHash == "" {
		return TelegramAccountUpsert{}, ErrTelegramInvalidData
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return TelegramAccountUpsert{}, ErrTelegramInvalidData
	}
	authDate, err := strconv.ParseInt(rawAuthDate, 10, 64)
	if err != nil || authDate <= 0 {
		return TelegramAccountUpsert{}, ErrTelegramInvalidData
	}

	return TelegramAccountUpsert{
		TelegramID:   id,
		Username:     strings.TrimSpace(fields["username"]),
		FirstName:    strings.TrimSpace(fields["first_name"]),
		LastName:     strings.TrimSpace(fields["last_name"]),
		PhotoURL:     strings.TrimSpace(fields["photo_url"]),
		LastAuthDate: authDate,
		LastLoginAt:  time.Unix(authDate, 0).UTC(),
	}, nil
}

func verifyTelegramHash(fields map[string]string, botToken string) bool {
	rawHash := strings.TrimSpace(fields["hash"])
	if rawHash == "" || botToken == "" {
		return false
	}
	providedHash, err := hex.DecodeString(rawHash)
	if err != nil {
		return false
	}

	dataCheckString := buildDataCheckString(fields)

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	_, _ = mac.Write([]byte(dataCheckString))
	expected := mac.Sum(nil)

	return hmac.Equal(providedHash, expected)
}

func buildDataCheckString(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if key == "hash" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+fields[key])
	}
	return strings.Join(lines, "\n")
}
