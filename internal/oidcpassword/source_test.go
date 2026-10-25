package oidcpassword

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeIdP — минимальный IdP: discovery + token endpoint (password и refresh_token).
type fakeIdP struct {
	srv        *httptest.Server
	now        func() time.Time
	ttl        time.Duration
	passwordN  atomic.Int32
	refreshN   atomic.Int32
	refreshBad atomic.Bool // отвечать invalid_grant на refresh
	counter    atomic.Int32
}

func newFakeIdP(t *testing.T, now func() time.Time, ttl time.Duration) *fakeIdP {
	t.Helper()
	f := &fakeIdP{now: now, ttl: ttl}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":         f.srv.URL,
			"token_endpoint": f.srv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("grant_type") {
		case "password":
			f.passwordN.Add(1)
			if r.Form.Get("username") != "alice" || r.Form.Get("password") != "s3cret" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "Invalid user credentials"})
				return
			}
		case "refresh_token":
			f.refreshN.Add(1)
			if f.refreshBad.Load() || r.Form.Get("refresh_token") == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		n := f.counter.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id_token":      makeJWT(f.now().Add(f.ttl).Unix(), n),
			"refresh_token": fmt.Sprintf("refresh-%d", n),
			"expires_in":    int(f.ttl.Seconds()),
		})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func makeJWT(exp int64, n int32) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "none"}) + "." + enc(map[string]any{"exp": exp, "n": n}) + ".sig"
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func newSource(t *testing.T, idp *fakeIdP, clk *clock, pass string) *Source {
	s := New(Config{IssuerURL: idp.srv.URL, ClientID: "kubernetes", Username: "alice", Password: pass})
	s.now = clk.Now
	return s
}

func TestPasswordGrantThenCache(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "s3cret")

	tok1, exp, err := s.Token(context.Background())
	if err != nil || tok1 == "" {
		t.Fatalf("token: %v", err)
	}
	if !exp.Equal(clk.Now().Add(5 * time.Minute)) {
		t.Errorf("expiry %v не равен exp из JWT", exp)
	}
	tok2, _, _ := s.Token(context.Background())
	if tok1 != tok2 || idp.passwordN.Load() != 1 {
		t.Fatalf("второй вызов должен браться из кэша (password=%d)", idp.passwordN.Load())
	}
}

func TestRefreshBeforeExpiry(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "s3cret")
	first, _, _ := s.Token(context.Background())

	clk.Add(5*time.Minute - 10*time.Second) // внутри окна refreshSkew
	second, _, err := s.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Error("токен должен обновиться до истечения")
	}
	if idp.refreshN.Load() != 1 || idp.passwordN.Load() != 1 || s.LastGrant() != "refresh" {
		t.Errorf("ждали 1 password + 1 refresh, а было password=%d refresh=%d grant=%s",
			idp.passwordN.Load(), idp.refreshN.Load(), s.LastGrant())
	}
}

func TestRevokedRefreshFallsBackToPassword(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "s3cret")
	_, _, _ = s.Token(context.Background())

	idp.refreshBad.Store(true)
	clk.Add(10 * time.Minute)
	if _, _, err := s.Token(context.Background()); err != nil {
		t.Fatalf("при отозванном refresh_token ждали повторный вход по паролю: %v", err)
	}
	if idp.passwordN.Load() != 2 || s.LastGrant() != "password" {
		t.Errorf("password=%d grant=%s", idp.passwordN.Load(), s.LastGrant())
	}
}

func TestWrongPasswordErrorHasNoSecrets(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "wrong-password-123")

	_, _, err := s.Token(context.Background())
	if err == nil {
		t.Fatal("ожидали ошибку")
	}
	for _, leak := range []string{"wrong-password-123", "alice"} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("ошибка раскрывает %q: %v", leak, err)
		}
	}
}

func TestConcurrentCallsHitIdPOnce(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "s3cret")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := s.Token(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if idp.passwordN.Load() != 1 {
		t.Fatalf("в IdP ушло %d запросов, ждали 1", idp.passwordN.Load())
	}
}

func TestRoundTripperSetsBearer(t *testing.T) {
	clk := &clock{t: time.Unix(1_800_000_000, 0)}
	idp := newFakeIdP(t, clk.Now, 5*time.Minute)
	s := newSource(t, idp, clk, "s3cret")

	var got string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	}))
	defer api.Close()

	c := &http.Client{Transport: s.RoundTripper(nil)}
	resp, err := c.Get(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	want, _, _ := s.Token(context.Background())
	if got != "Bearer "+want {
		t.Errorf("Authorization=%q", got)
	}
}

func TestParseOIDCLoginExec(t *testing.T) {
	p, ok, err := ParseOIDCLoginExec("kubectl", []string{
		"oidc-login", "get-token",
		"--oidc-issuer-url=https://isso.example/auth/realms/mts",
		"--oidc-client-id=kubernetes", "--grant-type=password",
	})
	if err != nil || !ok || p.IssuerURL != "https://isso.example/auth/realms/mts" || p.ClientID != "kubernetes" {
		t.Fatalf("p=%+v ok=%v err=%v", p, ok, err)
	}
	// значения отдельным аргументом
	p, ok, err = ParseOIDCLoginExec("kubectl", []string{"oidc-login", "get-token", "--oidc-issuer-url", "https://i", "--oidc-client-id", "c", "--grant-type", "password"})
	if err != nil || !ok || p.IssuerURL != "https://i" || p.ClientID != "c" {
		t.Fatalf("p=%+v ok=%v err=%v", p, ok, err)
	}
	// authcode по умолчанию — не поддерживается
	if _, ok, err := ParseOIDCLoginExec("kubectl", []string{"oidc-login", "get-token", "--oidc-issuer-url=https://i", "--oidc-client-id=c"}); !ok || err == nil {
		t.Errorf("authcode должен давать ошибку, ok=%v err=%v", ok, err)
	}
	// другой exec — не наш
	if _, ok, err := ParseOIDCLoginExec("aws", []string{"eks", "get-token"}); ok || err != nil {
		t.Errorf("aws не oidc-login: ok=%v err=%v", ok, err)
	}
}
