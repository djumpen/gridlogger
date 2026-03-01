package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

type telegramStoreStub struct {
	accounts map[int64]TelegramAccount
}

func (s *telegramStoreStub) UpsertTelegramAccount(_ context.Context, in TelegramAccountUpsert) (TelegramAccount, bool, error) {
	if s.accounts == nil {
		s.accounts = map[int64]TelegramAccount{}
	}
	if existing, ok := s.accounts[in.TelegramID]; ok {
		if in.LastAuthDate <= existing.LastAuthDate {
			return TelegramAccount{}, true, nil
		}
	}

	account := TelegramAccount{
		TelegramID:   in.TelegramID,
		Username:     in.Username,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		PhotoURL:     in.PhotoURL,
		LastAuthDate: in.LastAuthDate,
		LastLoginAt:  in.LastLoginAt,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	s.accounts[in.TelegramID] = account
	return account, false, nil
}

func (s *telegramStoreStub) GetTelegramAccountByID(_ context.Context, telegramID int64) (TelegramAccount, bool, error) {
	account, ok := s.accounts[telegramID]
	return account, ok, nil
}

func TestTelegramAuthServiceAuthenticateSuccess(t *testing.T) {
	store := &telegramStoreStub{}
	svc := NewTelegramAuthService(store, "123456:ABCDEF", 24*time.Hour)

	authDate := int64(1710000000)
	fields := map[string]string{
		"id":         "900001",
		"first_name": "Dmytro",
		"last_name":  "S",
		"username":   "djumpen",
		"auth_date":  "1710000000",
	}
	fields["hash"] = signTelegramFields(fields, "123456:ABCDEF")
	svc.nowFn = func() time.Time { return time.Unix(authDate+60, 0).UTC() }

	account, err := svc.Authenticate(context.Background(), fields)
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if account.TelegramID != 900001 {
		t.Fatalf("unexpected telegram id: %d", account.TelegramID)
	}
	if account.LastAuthDate != authDate {
		t.Fatalf("unexpected auth date: %d", account.LastAuthDate)
	}
}

func TestTelegramAuthServiceReplayRejected(t *testing.T) {
	store := &telegramStoreStub{}
	svc := NewTelegramAuthService(store, "123456:ABCDEF", 24*time.Hour)
	svc.nowFn = func() time.Time { return time.Unix(1710000100, 0).UTC() }

	fields := map[string]string{
		"id":         "900001",
		"first_name": "Dmytro",
		"auth_date":  "1710000000",
	}
	fields["hash"] = signTelegramFields(fields, "123456:ABCDEF")

	if _, err := svc.Authenticate(context.Background(), fields); err != nil {
		t.Fatalf("first Authenticate error: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), fields); !errors.Is(err, ErrTelegramReplay) {
		t.Fatalf("expected replay error, got: %v", err)
	}
}

func TestTelegramAuthServiceStaleRejected(t *testing.T) {
	store := &telegramStoreStub{}
	svc := NewTelegramAuthService(store, "123456:ABCDEF", 24*time.Hour)
	svc.nowFn = func() time.Time { return time.Unix(1710087000, 0).UTC() }

	fields := map[string]string{
		"id":         "900001",
		"first_name": "Dmytro",
		"auth_date":  "1710000000",
	}
	fields["hash"] = signTelegramFields(fields, "123456:ABCDEF")

	_, err := svc.Authenticate(context.Background(), fields)
	if !errors.Is(err, ErrTelegramAuthStale) {
		t.Fatalf("expected stale error, got: %v", err)
	}
}

func TestTelegramAuthServiceHashMismatchRejected(t *testing.T) {
	store := &telegramStoreStub{}
	svc := NewTelegramAuthService(store, "123456:ABCDEF", 24*time.Hour)
	svc.nowFn = func() time.Time { return time.Unix(1710000100, 0).UTC() }

	fields := map[string]string{
		"id":         "900001",
		"first_name": "Dmytro",
		"auth_date":  "1710000000",
		"hash":       "00",
	}

	_, err := svc.Authenticate(context.Background(), fields)
	if !errors.Is(err, ErrTelegramHashMismatch) {
		t.Fatalf("expected hash mismatch, got: %v", err)
	}
}

func signTelegramFields(fields map[string]string, botToken string) string {
	copyFields := map[string]string{}
	for k, v := range fields {
		copyFields[k] = v
	}
	delete(copyFields, "hash")
	dataCheckString := buildDataCheckString(copyFields)

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	_, _ = mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil))
}
