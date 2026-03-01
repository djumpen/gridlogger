package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrSessionDisabled = errors.New("session auth is disabled")
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("token expired")
)

type SessionClaims struct {
	TelegramID int64
	IssuedAt   time.Time
	ExpiresAt  time.Time
}

type SessionService struct {
	secret []byte
	issuer string
	ttl    time.Duration
	nowFn  func() time.Time
}

func NewSessionService(secret, issuer string, ttl time.Duration) *SessionService {
	return &SessionService{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *SessionService) Enabled() bool {
	return len(s.secret) > 0
}

func (s *SessionService) IssueToken(telegramID int64) (string, error) {
	if !s.Enabled() {
		return "", ErrSessionDisabled
	}
	if telegramID <= 0 {
		return "", ErrInvalidToken
	}

	now := s.nowFn().UTC()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := map[string]any{
		"iss": s.issuer,
		"sub": strconv.FormatInt(telegramID, 10),
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
	}

	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headB64 := base64.RawURLEncoding.EncodeToString(headerRaw)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsRaw)
	signingInput := headB64 + "." + claimsB64

	sig := signHMACSHA256([]byte(signingInput), s.secret)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}

func (s *SessionService) ParseToken(token string) (SessionClaims, error) {
	if !s.Enabled() {
		return SessionClaims{}, ErrSessionDisabled
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return SessionClaims{}, ErrInvalidToken
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return SessionClaims{}, ErrInvalidToken
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return SessionClaims{}, ErrInvalidToken
	}
	providedSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return SessionClaims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := signHMACSHA256([]byte(signingInput), s.secret)
	if !hmac.Equal(providedSig, expectedSig) {
		return SessionClaims{}, ErrInvalidToken
	}

	var header map[string]string
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return SessionClaims{}, ErrInvalidToken
	}
	if header["alg"] != "HS256" || header["typ"] != "JWT" {
		return SessionClaims{}, ErrInvalidToken
	}

	var payload struct {
		Issuer string `json:"iss"`
		Sub    string `json:"sub"`
		Iat    int64  `json:"iat"`
		Exp    int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return SessionClaims{}, ErrInvalidToken
	}
	if payload.Issuer != s.issuer {
		return SessionClaims{}, ErrInvalidToken
	}
	telegramID, err := strconv.ParseInt(payload.Sub, 10, 64)
	if err != nil || telegramID <= 0 {
		return SessionClaims{}, ErrInvalidToken
	}

	now := s.nowFn().UTC()
	expiresAt := time.Unix(payload.Exp, 0).UTC()
	if !expiresAt.After(now) {
		return SessionClaims{}, ErrExpiredToken
	}

	issuedAt := time.Unix(payload.Iat, 0).UTC()
	if issuedAt.After(now.Add(60 * time.Second)) {
		return SessionClaims{}, fmt.Errorf("%w: issued in future", ErrInvalidToken)
	}

	return SessionClaims{
		TelegramID: telegramID,
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
	}, nil
}

func signHMACSHA256(input []byte, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(input)
	return mac.Sum(nil)
}
