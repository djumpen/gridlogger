package service

import (
	"errors"
	"testing"
	"time"
)

func TestSessionServiceIssueAndParse(t *testing.T) {
	svc := NewSessionService("01234567890123456789012345678901", "gridlogger", 2*time.Hour)
	base := time.Unix(1710000000, 0).UTC()
	svc.nowFn = func() time.Time { return base }

	token, err := svc.IssueToken(12345)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}

	svc.nowFn = func() time.Time { return base.Add(10 * time.Minute) }
	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != 12345 {
		t.Fatalf("unexpected user id: %d", claims.UserID)
	}
}

func TestSessionServiceExpiredToken(t *testing.T) {
	svc := NewSessionService("01234567890123456789012345678901", "gridlogger", 1*time.Hour)
	base := time.Unix(1710000000, 0).UTC()
	svc.nowFn = func() time.Time { return base }

	token, err := svc.IssueToken(12345)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}

	svc.nowFn = func() time.Time { return base.Add(2 * time.Hour) }
	_, err = svc.ParseToken(token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got: %v", err)
	}
}
