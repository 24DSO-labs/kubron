// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Session — веб-сессия браузера: привязывает cookie к пользователю и CSRF-токену.
type Session struct {
	// ID — публичный идентификатор сессии (не секрет, не токен): для привязки WebSocket-соединений.
	ID      string
	UserID  string
	CSRF    string
	Expires time.Time
}

// Sessions — реестр веб-сессий. Идентификатор сессии (256 бит) выдаётся браузеру в cookie,
// а в памяти хранится только его SHA-256: дамп памяти или лога не даёт действующий токен.
//
// Ограничение до этапа 2: ядро хранит одну общую сессию пользователя, поэтому в режиме
// Exclusive новый вход отзывает все прочие веб-сессии (иначе старый браузер получил бы права
// нового пользователя). С появлением Session в ядре (этап 2) Exclusive отключается.
type Sessions struct {
	// Exclusive — не более одной активной веб-сессии.
	Exclusive bool
	// OnEnd вызывается (вне блокировки) для каждой завершённой сессии: выход, истечение, вытеснение.
	OnEnd func(sessionID string)

	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[[sha256.Size]byte]*Session
}

// NewSessions создаёт реестр; ttl — абсолютный срок жизни сессии (простой считает ядро).
func NewSessions(ttl time.Duration, exclusive bool) *Sessions {
	return &Sessions{Exclusive: exclusive, ttl: ttl, now: time.Now, items: map[[sha256.Size]byte]*Session{}}
}

func tokenKey(token string) [sha256.Size]byte { return sha256.Sum256([]byte(token)) }

func sessionID(k [sha256.Size]byte) string { return hex.EncodeToString(k[:8]) }

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("server: нет источника случайности: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Create открывает сессию для пользователя и возвращает токен для cookie.
func (s *Sessions) Create(userID string) (token string, sess Session) {
	s.mu.Lock()
	var ended []string
	if s.Exclusive {
		for k := range s.items {
			ended = append(ended, sessionID(k))
		}
		clear(s.items)
	}
	ended = append(ended, s.sweepLocked()...)
	token = randomToken()
	k := tokenKey(token)
	sess = Session{ID: sessionID(k), UserID: userID, CSRF: randomToken(), Expires: s.now().Add(s.ttl)}
	cp := sess
	s.items[k] = &cp
	s.mu.Unlock()
	s.notify(ended)
	return token, sess
}

func (s *Sessions) notify(ids []string) {
	if s.OnEnd == nil {
		return
	}
	for _, id := range ids {
		s.OnEnd(id)
	}
}

// Get возвращает действующую сессию по токену из cookie.
func (s *Sessions) Get(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	s.mu.Lock()
	k := tokenKey(token)
	it, ok := s.items[k]
	if !ok {
		s.mu.Unlock()
		return Session{}, false
	}
	if !s.now().Before(it.Expires) {
		delete(s.items, k)
		s.mu.Unlock()
		s.notify([]string{sessionID(k)})
		return Session{}, false
	}
	out := *it
	s.mu.Unlock()
	return out, true
}

// Delete закрывает сессию.
func (s *Sessions) Delete(token string) {
	k := tokenKey(token)
	s.mu.Lock()
	_, had := s.items[k]
	delete(s.items, k)
	s.mu.Unlock()
	if had {
		s.notify([]string{sessionID(k)})
	}
}

// Len — число живых сессий (для тестов и метрик).
func (s *Sessions) Len() int {
	s.mu.Lock()
	ended := s.sweepLocked()
	n := len(s.items)
	s.mu.Unlock()
	s.notify(ended)
	return n
}

func (s *Sessions) sweepLocked() (ended []string) {
	now := s.now()
	for k, it := range s.items {
		if !now.Before(it.Expires) {
			delete(s.items, k)
			ended = append(ended, sessionID(k))
		}
	}
	return ended
}

// csrfEqual сравнивает токены за постоянное время.
func csrfEqual(a, b string) bool {
	return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

const (
	cookieName       = "kubron_session"
	secureCookieName = "__Host-kubron_session" // префикс __Host- требует Secure, Path=/ и отсутствия Domain
	csrfHeader       = "X-CSRF-Token"
)

// Guard — защита веб-API: cookie-сессии, CSRF-токен, источник запроса.
type Guard struct {
	Sessions *Sessions
	// TrustProxy: доверять X-Forwarded-Proto от обратного прокси (TLS завершается на нём).
	TrustProxy bool
	// Hub — WebSocket-хаб событий; nil отключает /api/ws.
	Hub *Hub
}

func (g *Guard) secure(r *http.Request) bool {
	return r.TLS != nil || (g.TrustProxy && r.Header.Get("X-Forwarded-Proto") == "https")
}

func (g *Guard) token(r *http.Request) string {
	for _, n := range []string{secureCookieName, cookieName} {
		if c, err := r.Cookie(n); err == nil && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func (g *Guard) setCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	c := &http.Cookie{
		Name: cookieName, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: g.secure(r),
	}
	if c.Secure {
		c.Name = secureCookieName
	}
	if ttl < 0 {
		c.MaxAge = -1
		c.Value = ""
	} else {
		c.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, c)
}

func (g *Guard) clearCookie(w http.ResponseWriter, r *http.Request) {
	g.setCookie(w, r, "", -1)
}
