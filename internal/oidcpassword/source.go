// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package oidcpassword — нативное получение и обновление OIDC id_token по grant_type=password.
//
// Нужно серверной (веб) версии Kubron: kubeconfig с exec-плагином `kubectl oidc-login`
// на сервере запускать нельзя (RCE, нет TTY), поэтому сервер сам ходит в IdP.
// Только stdlib; grant password в OAuth считается устаревшим, но именно его используют
// существующие kubeconfig'и. Другие grant'ы не поддерживаются.
package oidcpassword

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config — параметры входа. Пароль хранится в памяти процесса (в хранилище — зашифрованным, ADR-0008).
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string // необязателен (public client)
	Username     string
	Password     string
	Scopes       []string // по умолчанию ["openid"]
	HTTPClient   *http.Client
}

// Source выдаёт актуальный id_token, обновляя его по refresh_token и, если не вышло,
// повторяя вход по паролю. Безопасен для параллельных вызовов: в IdP уходит один запрос.
type Source struct {
	cfg Config
	now func() time.Time

	mu       sync.Mutex
	tokenEP  string
	idToken  string
	expiry   time.Time
	refresh  string
	lastAuth string // "password" | "refresh" — для диагностики
}

// refreshSkew — за сколько до истечения токен считается устаревшим.
const refreshSkew = 30 * time.Second

func New(cfg Config) *Source {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid"}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Source{cfg: cfg, now: time.Now}
}

// Token возвращает id_token и момент его истечения.
func (s *Source) Token(ctx context.Context) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idToken != "" && s.now().Add(refreshSkew).Before(s.expiry) {
		return s.idToken, s.expiry, nil
	}
	if s.tokenEP == "" {
		ep, err := s.discover(ctx)
		if err != nil {
			return "", time.Time{}, err
		}
		s.tokenEP = ep
	}
	if s.refresh != "" {
		if err := s.request(ctx, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {s.refresh},
		}, "refresh"); err == nil {
			return s.idToken, s.expiry, nil
		}
		s.refresh = "" // refresh_token отозван/просрочен — входим заново по паролю
	}
	if err := s.request(ctx, url.Values{
		"grant_type": {"password"},
		"username":   {s.cfg.Username},
		"password":   {s.cfg.Password},
		"scope":      {strings.Join(s.cfg.Scopes, " ")},
	}, "password"); err != nil {
		return "", time.Time{}, err
	}
	return s.idToken, s.expiry, nil
}

// LastGrant сообщает, каким способом получен текущий токен (для тестов и логов).
func (s *Source) LastGrant() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAuth
}

func (s *Source) discover(ctx context.Context) (string, error) {
	u := strings.TrimRight(s.cfg.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oidc discovery: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oidc discovery: HTTP %d", resp.StatusCode)
	}
	var doc struct {
		Issuer        string `json:"issuer"`
		TokenEndpoint string `json:"token_endpoint"`
	}
	if err := json.Unmarshal(body, &doc); err != nil || doc.TokenEndpoint == "" {
		return "", errors.New("oidc discovery: в ответе нет token_endpoint")
	}
	if doc.Issuer != "" && strings.TrimRight(doc.Issuer, "/") != strings.TrimRight(s.cfg.IssuerURL, "/") {
		return "", fmt.Errorf("oidc discovery: issuer %q не совпадает с настроенным %q", doc.Issuer, s.cfg.IssuerURL)
	}
	return doc.TokenEndpoint, nil
}

type tokenResponse struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func (s *Source) request(ctx context.Context, form url.Values, grant string) error {
	form.Set("client_id", s.cfg.ClientID)
	if s.cfg.ClientSecret != "" {
		form.Set("client_secret", s.cfg.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenEP, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("oidc %s: %w", grant, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var tr tokenResponse
	_ = json.Unmarshal(body, &tr)
	if resp.StatusCode != http.StatusOK {
		// Пароль и токены в сообщение не попадают.
		return fmt.Errorf("oidc %s: HTTP %d %s %s", grant, resp.StatusCode, tr.Error, tr.ErrorDesc)
	}
	if tr.IDToken == "" {
		return fmt.Errorf("oidc %s: в ответе нет id_token (нужен scope openid)", grant)
	}
	exp, err := jwtExpiry(tr.IDToken)
	if err != nil {
		if tr.ExpiresIn <= 0 {
			return fmt.Errorf("oidc %s: не удалось определить срок id_token: %w", grant, err)
		}
		exp = s.now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	s.idToken, s.expiry, s.lastAuth = tr.IDToken, exp, grant
	if tr.RefreshToken != "" {
		s.refresh = tr.RefreshToken
	}
	return nil
}

// jwtExpiry читает claim exp без проверки подписи (подпись проверяет kube-apiserver).
func jwtExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, errors.New("не JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	var c struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &c); err != nil || c.Exp == 0 {
		return time.Time{}, errors.New("в токене нет exp")
	}
	return time.Unix(c.Exp, 0), nil
}

// RoundTripper подставляет актуальный id_token в каждый запрос к kube-apiserver.
// Подключается через rest.Config.Wrap(src.RoundTripper).
func (s *Source) RoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		tok, _, err := s.Token(r.Context())
		if err != nil {
			return nil, err
		}
		r2 := r.Clone(r.Context())
		r2.Header.Set("Authorization", "Bearer "+tok)
		return next.RoundTrip(r2)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
