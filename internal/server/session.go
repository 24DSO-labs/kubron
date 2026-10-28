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

	"github.com/VadimIl224/kubron/internal/termbuf"
)

// Session — веб-сессия браузера: привязывает cookie к пользователю и CSRF-токену.
type Session struct {
	// ID — публичный идентификатор сессии (не секрет, не токен): для привязки WebSocket-соединений.
	ID     string
	UserID string
	CSRF   string
	// View — вид ядра этой сессии (nil — общий экземпляр диспетчера). Освобождается при завершении сессии.
	View    any
	Expires time.Time
}

// Sessions — реестр веб-сессий. Идентификатор сессии (256 бит) выдаётся браузеру в cookie,
// а в памяти хранится только его SHA-256: дамп памяти или лога не даёт действующий токен.
//
// Каждая веб-сессия владеет видом ядра (Session.View): у него своя сессия пользователя, права и
// ресурсы. Режим Exclusive (не более одной веб-сессии, новый вход вытесняет прежнюю) нужен только
// там, где ядро общее на всех (тесты, одиночный режим).
type Sessions struct {
	// Exclusive — не более одной активной веб-сессии.
	Exclusive bool
	// OnEnd вызывается (вне блокировки) для каждой завершённой сессии: выход, истечение, вытеснение.
	OnEnd func(sessionID string)
	// Release освобождает вид ядра завершённой сессии (вне блокировки); вызывается до OnEnd.
	Release func(view any)

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
func (s *Sessions) Create(userID string, view any) (token string, sess Session) {
	s.mu.Lock()
	var ended []Session
	if s.Exclusive {
		for _, it := range s.items {
			ended = append(ended, *it)
		}
		clear(s.items)
	}
	ended = append(ended, s.sweepLocked()...)
	token = randomToken()
	k := tokenKey(token)
	sess = Session{ID: sessionID(k), UserID: userID, CSRF: randomToken(), View: view, Expires: s.now().Add(s.ttl)}
	cp := sess
	s.items[k] = &cp
	s.mu.Unlock()
	s.notify(ended)
	return token, sess
}

func (s *Sessions) notify(ended []Session) {
	for _, e := range ended {
		if s.Release != nil && e.View != nil {
			s.Release(e.View)
		}
		if s.OnEnd != nil {
			s.OnEnd(e.ID)
		}
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
		s.notify([]Session{*it})
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
	it, had := s.items[k]
	delete(s.items, k)
	s.mu.Unlock()
	if had {
		s.notify([]Session{*it})
	}
}

// Reap завершает сессии, у которых истёк срок или чей вид ядра больше не жив (alive == false:
// ядро закрыло пользовательскую сессию из-за простоя, отключения пользователя и т. п.). Без фоновой
// уборки такие сессии держали бы ссылки пула подключений, пока кто-нибудь не обратится к серверу.
// Возвращает число завершённых сессий.
func (s *Sessions) Reap(alive func(view any) bool) int {
	s.mu.Lock()
	ended := s.sweepLocked()
	if alive != nil {
		for k, it := range s.items {
			if it.View != nil && !alive(it.View) {
				delete(s.items, k)
				ended = append(ended, *it)
			}
		}
	}
	s.mu.Unlock()
	s.notify(ended)
	return len(ended)
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

func (s *Sessions) sweepLocked() (ended []Session) {
	now := s.now()
	for k, it := range s.items {
		if !now.Before(it.Expires) {
			delete(s.items, k)
			ended = append(ended, *it)
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
	// Terminals — реестр веб-терминалов; nil отключает /api/terminal/{id}.
	Terminals *termbuf.Registry
	// NewView создаёт вид ядра для нового входа (source — адрес клиента); nil — все сессии работают
	// с общим экземпляром диспетчера (тесты). Освобождение — Sessions.Release.
	NewView func(source string) any
	// Bind сообщает виду идентификатор созданной для него веб-сессии (адресат событий); nil — не нужно.
	Bind func(view any, sessionID string)

	termConns termConns
}

// CloseSession закрывает все WebSocket сессии (события и терминалы). Подключается к Sessions.OnEnd.
func (g *Guard) CloseSession(sessionID string) {
	if g.Hub != nil {
		g.Hub.CloseSession(sessionID)
	}
	g.termConns.closeSession(sessionID)
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
