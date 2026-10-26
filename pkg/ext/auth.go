package ext

import (
	"context"
	"net/url"
)

// Kind — способ входа, который предоставляет Authenticator.
type Kind string

const (
	// KindPassword — форма «логин и пароль» в самом Kubron (например, локальные пользователи, LDAP).
	KindPassword Kind = "password"
	// KindRedirect — вход через внешний провайдер с перенаправлением браузера (OIDC, SAML).
	KindRedirect Kind = "redirect"
)

// Identity — подтверждённая личность пользователя. Роль в Kubron назначает ядро (по Groups/Claims
// и своим правилам), а не аутентификатор.
type Identity struct {
	Subject     string         // стабильный идентификатор у провайдера (claim sub)
	Username    string         // логин для отображения и журнала
	DisplayName string         // имя для отображения
	Email       string         //
	Groups      []string       // группы/роли провайдера
	Claims      map[string]any // прочие утверждения провайдера (только для правил сопоставления)
}

// Authenticator — способ входа в Kubron. Реализация обязана также реализовать
// PasswordAuthenticator или RedirectAuthenticator в соответствии с Kind().
type Authenticator interface {
	// ID — уникальное короткое имя ("local", "oidc"); используется в URL и конфигурации.
	ID() string
	Kind() Kind
}

// PasswordAuthenticator проверяет логин и пароль.
type PasswordAuthenticator interface {
	Authenticator
	Authenticate(ctx context.Context, username, password string) (*Identity, error)
}

// RedirectStart — результат начала входа с перенаправлением.
type RedirectStart struct {
	// URL — куда отправить браузер.
	URL string
	// State — непрозрачные данные аутентификатора (state, nonce, PKCE verifier). Ядро хранит их на
	// сервере (в сессии/БД, чтобы работало при нескольких репликах) и вернёт в CompleteLogin.
	State []byte
}

// RedirectAuthenticator выполняет вход через внешний провайдер (authorization code + PKCE для OIDC).
// Сам аутентификатор не хранит состояние между вызовами: всё нужное лежит в RedirectStart.State.
type RedirectAuthenticator interface {
	Authenticator
	// BeginLogin вызывается при нажатии «Войти через SSO». redirectURI — адрес обратного вызова Kubron.
	BeginLogin(ctx context.Context, redirectURI string) (*RedirectStart, error)
	// CompleteLogin вызывается на обратном вызове с параметрами запроса и сохранённым State.
	CompleteLogin(ctx context.Context, state []byte, params url.Values) (*Identity, error)
}
