// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// directives разбирает CSP в карту «директива → значение».
func directives(csp string) map[string]string {
	out := map[string]string{}
	for _, d := range strings.Split(csp, ";") {
		d = strings.TrimSpace(d)
		name, value, _ := strings.Cut(d, " ")
		out[name] = value
	}
	return out
}

func ok200() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestContentSecurityPolicyDirectives(t *testing.T) {
	d := directives(contentSecurityPolicy("kubron.example.com"))
	want := map[string]string{
		"default-src":     "'self'",
		"script-src":      "'self'",
		"style-src":       "'self' 'unsafe-inline'",
		"img-src":         "'self' data:",
		"font-src":        "'self'",
		"connect-src":     "'self' ws://kubron.example.com wss://kubron.example.com",
		"worker-src":      "'self' blob:",
		"object-src":      "'none'",
		"base-uri":        "'self'",
		"form-action":     "'self'",
		"frame-ancestors": "'none'",
	}
	for name, value := range want {
		if d[name] != value {
			t.Errorf("%s: %q, want %q", name, d[name], value)
		}
	}
	for _, name := range []string{"script-src", "default-src"} {
		if strings.Contains(d[name], "unsafe-inline") || strings.Contains(d[name], "unsafe-eval") {
			t.Errorf("%s must not allow unsafe-inline/unsafe-eval: %q", name, d[name])
		}
	}
	if strings.Contains(contentSecurityPolicy("x"), "unsafe-eval") {
		t.Error("unsafe-eval must never appear in the policy")
	}
}

// Host приходит от клиента и попадает в заголовок: недопустимые значения не должны дописывать директивы.
func TestContentSecurityPolicyRejectsHostInjection(t *testing.T) {
	for _, host := range []string{
		"evil.com; script-src *", "a b", "x'y", "evil.com,other", "", "http://evil.com", "host\r\nX: y", "-bad-.com",
	} {
		csp := contentSecurityPolicy(host)
		if d := directives(csp); d["connect-src"] != "'self'" || d["script-src"] != "'self'" {
			t.Errorf("host %q leaked into the policy: %s", host, csp)
		}
	}
	for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", "kubron.example.com", "a.b-c.example"} {
		if !strings.Contains(contentSecurityPolicy(host), "ws://"+host) {
			t.Errorf("valid host %q must be allowed for WebSocket", host)
		}
	}
}

func TestSecurityHeadersOnEveryPath(t *testing.T) {
	h := securityHeaders(ok200(), false, handlerOptions{})
	for _, path := range []string{"/", "/index.html", "/assets/app-abc.js", "/api/rpc/Login", "/api/session", "/api/unknown", "/healthz", "/missing/route"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "kubron.example.com"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		hd := rec.Header()
		for name, want := range map[string]string{
			"X-Content-Type-Options":       "nosniff",
			"X-Frame-Options":              "DENY",
			"Referrer-Policy":              "no-referrer",
			"Cross-Origin-Opener-Policy":   "same-origin",
			"Cross-Origin-Resource-Policy": "same-origin",
		} {
			if got := hd.Get(name); got != want {
				t.Errorf("%s %s: %q, want %q", path, name, got, want)
			}
		}
		if csp := hd.Get("Content-Security-Policy"); csp != contentSecurityPolicy("kubron.example.com") {
			t.Errorf("%s CSP: %q", path, csp)
		}
		if hd.Get("Content-Security-Policy-Report-Only") != "" {
			t.Errorf("%s: Report-Only must not be set by default", path)
		}
		pp := hd.Get("Permissions-Policy")
		for _, f := range []string{"camera=()", "microphone=()", "geolocation=()"} {
			if !strings.Contains(pp, f) {
				t.Errorf("%s Permissions-Policy %q misses %s", path, pp, f)
			}
		}
		if hd.Get("Strict-Transport-Security") != "" {
			t.Errorf("%s: HSTS over plain HTTP", path)
		}
	}
}

// Заголовки ставятся и на ответах-ошибках, которые формирует сам обработчик.
func TestSecurityHeadersOnErrorResponses(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusInternalServerError, http.StatusServiceUnavailable} {
		h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", status)
		}), false, handlerOptions{})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		if rec.Code != status || rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("status %d: code=%d headers=%v", status, rec.Code, rec.Header())
		}
	}
}

func TestCSPReportOnlyMode(t *testing.T) {
	h := securityHeaders(ok200(), false, handlerOptions{cspReportOnly: true})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Security-Policy") != "" {
		t.Error("enforcing header must not be set in report-only mode")
	}
	if rec.Header().Get("Content-Security-Policy-Report-Only") == "" {
		t.Error("report-only header missing")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options still protects against framing in report-only mode")
	}
}

func TestHSTSOnlyOnSecureNonLoopbackRequests(t *testing.T) {
	tlsState := &tls.ConnectionState{}
	for _, c := range []struct {
		name       string
		host       string
		tls        bool
		xfp        string
		trustProxy bool
		want       bool
	}{
		{"plain http", "kubron.example.com", false, "", false, false},
		{"direct TLS", "kubron.example.com", true, "", false, true},
		{"proxy says https, trusted", "kubron.example.com", false, "https", true, true},
		{"proxy says https, NOT trusted", "kubron.example.com", false, "https", false, false},
		{"proxy says http, trusted", "kubron.example.com", false, "http", true, false},
		{"TLS but localhost", "localhost:8443", true, "", false, false},
		{"TLS but 127.0.0.1", "127.0.0.1:8443", true, "", false, false},
		{"TLS but IPv6 loopback", "[::1]:8443", true, "", false, false},
		{"trusted proxy https, localhost", "localhost", false, "https", true, false},
		{"TLS, public IP", "203.0.113.5:8443", true, "", false, true},
	} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = c.host
		if c.tls {
			req.TLS = tlsState
		}
		if c.xfp != "" {
			req.Header.Set("X-Forwarded-Proto", c.xfp)
		}
		rec := httptest.NewRecorder()
		securityHeaders(ok200(), c.trustProxy, handlerOptions{}).ServeHTTP(rec, req)
		got := rec.Header().Get("Strict-Transport-Security")
		if (got != "") != c.want {
			t.Errorf("%s: HSTS=%q, want present=%v", c.name, got, c.want)
		}
		if c.want && !strings.HasPrefix(got, "max-age=") {
			t.Errorf("%s: HSTS value %q", c.name, got)
		}
	}
}
