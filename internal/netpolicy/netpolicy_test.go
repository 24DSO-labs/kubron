// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package netpolicy

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func mustAllow(t *testing.T, s string) []netip.Prefix {
	t.Helper()
	p, err := ParseAllowList(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func quiet(p *Policy) *Policy {
	p.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return p
}

func TestParseIPHost(t *testing.T) {
	cases := []struct {
		in   string
		want string // "" — не IP
	}{
		{"127.0.0.1", "127.0.0.1"},
		{"2130706433", "127.0.0.1"},
		{"0177.0.0.1", "127.0.0.1"},
		{"0x7f.0.0.1", "127.0.0.1"},
		{"0x7f000001", "127.0.0.1"},
		{"127.1", "127.0.0.1"},
		{"127.0.1", "127.0.0.1"},
		{"2852039166", "169.254.169.254"},
		{"0251.0376.0251.0376", "169.254.169.254"},
		{"127.0.0.1.", "127.0.0.1"},
		{"::ffff:127.0.0.1", "127.0.0.1"},
		{"[::1]", "::1"},
		{"fe80::1%eth0", "fe80::1"},
		{"0", "0.0.0.0"},
		{"example.com", ""},
		{"1.2.3.4.5", ""},
		{"08.0.0.1", ""},
		{"1_0.0.0.1", ""},
		{"0x", ""},
		{"256.0.0.1", ""},
		{"4294967296", ""},
		{"127.0.0.0x100", ""},
		{"", ""},
		{"::zz", ""},
	}
	for _, c := range cases {
		got, ok := ParseIPHost(c.in)
		if c.want == "" {
			if ok {
				t.Errorf("ParseIPHost(%q) = %v, want not an IP", c.in, got)
			}
			continue
		}
		if !ok || got.String() != c.want {
			t.Errorf("ParseIPHost(%q) = %v,%v, want %s", c.in, got, ok, c.want)
		}
	}
}

func TestVerdict(t *testing.T) {
	permissive := quiet(New("t", ProfilePermissive, nil))
	strict := quiet(New("t", ProfileStrict, nil))
	cases := []struct {
		ip         string
		permissive Reason
		strict     Reason
	}{
		{"8.8.8.8", "", ""},
		{"2606:4700:4700::1111", "", ""},
		{"127.0.0.1", ReasonLoopback, ReasonLoopback},
		{"127.255.255.254", ReasonLoopback, ReasonLoopback},
		{"::1", ReasonLoopback, ReasonLoopback},
		{"169.254.1.1", ReasonLinkLocal, ReasonLinkLocal},
		{"fe80::1", ReasonLinkLocal, ReasonLinkLocal},
		{"169.254.169.254", ReasonMetadata, ReasonMetadata},
		{"fd00:ec2::254", ReasonMetadata, ReasonMetadata},
		{"168.63.129.16", ReasonMetadata, ReasonMetadata},
		{"100.100.100.200", ReasonMetadata, ReasonMetadata},
		{"0.0.0.0", ReasonUnspecified, ReasonUnspecified},
		{"0.1.2.3", ReasonUnspecified, ReasonUnspecified},
		{"::", ReasonUnspecified, ReasonUnspecified},
		{"255.255.255.255", ReasonReserved, ReasonReserved},
		{"224.0.0.1", ReasonMulticast, ReasonMulticast},
		{"ff02::1", ReasonMulticast, ReasonMulticast},
		// частные диапазоны: только в strict
		{"10.1.2.3", "", ReasonPrivate},
		{"172.16.0.1", "", ReasonPrivate},
		{"172.31.255.255", "", ReasonPrivate},
		{"172.32.0.1", "", ""},
		{"192.168.1.1", "", ReasonPrivate},
		{"100.64.0.1", "", ReasonPrivate},
		{"100.127.255.255", "", ReasonPrivate},
		{"100.128.0.1", "", ""},
		{"fc00::1", "", ReasonPrivate},
		{"fdff::1", "", ReasonPrivate},
		{"fd00:ec2::253", "", ReasonPrivate},
		// IPv4 внутри IPv6
		{"::ffff:127.0.0.1", ReasonLoopback, ReasonLoopback},
		{"::ffff:169.254.169.254", ReasonMetadata, ReasonMetadata},
		{"::ffff:10.0.0.1", "", ReasonPrivate},
		{"64:ff9b::a9fe:a9fe", ReasonMetadata, ReasonMetadata},
		{"64:ff9b::7f00:1", ReasonLoopback, ReasonLoopback},
		{"2002:a9fe:a9fe::1", ReasonMetadata, ReasonMetadata},
		{"::ffff:0:a9fe:a9fe", ReasonMetadata, ReasonMetadata},
		{"::a9fe:a9fe", ReasonMetadata, ReasonMetadata},
		{"64:ff9b::808:808", "", ""},
	}
	for _, c := range cases {
		a := netip.MustParseAddr(c.ip)
		if got := permissive.verdict(a); got != c.permissive {
			t.Errorf("permissive %s: got %q, want %q", c.ip, got, c.permissive)
		}
		if got := strict.verdict(a); got != c.strict {
			t.Errorf("strict %s: got %q, want %q", c.ip, got, c.strict)
		}
	}
}

func TestAllowList(t *testing.T) {
	cases := []struct {
		name    string
		profile Profile
		allow   string
		ip      string
		blocked bool
	}{
		{"private lifted", ProfileStrict, "10.0.0.0/8", "10.1.1.1", false},
		{"other private stays", ProfileStrict, "10.0.0.0/8", "192.168.1.1", true},
		{"single address", ProfileStrict, "192.168.1.5", "192.168.1.5", false},
		{"single address neighbour", ProfileStrict, "192.168.1.5", "192.168.1.6", true},
		{"loopback lifted", ProfilePermissive, "127.0.0.0/8", "127.0.0.1", false},
		{"link-local lifted", ProfilePermissive, "169.254.0.0/16", "169.254.1.1", false},
		{"metadata never lifted", ProfilePermissive, "169.254.0.0/16", "169.254.169.254", true},
		{"metadata never lifted by 0/0", ProfilePermissive, "0.0.0.0/0", "169.254.169.254", true},
		{"mapped metadata never lifted", ProfilePermissive, "0.0.0.0/0", "::ffff:169.254.169.254", true},
		{"multicast never lifted", ProfilePermissive, "224.0.0.0/4", "224.0.0.1", true},
		{"unspecified never lifted", ProfilePermissive, "0.0.0.0/0", "0.0.0.0", true},
		{"nat64 of allowed private", ProfileStrict, "10.0.0.0/8", "64:ff9b::a01:101", false},
		{"nat64 of other private", ProfileStrict, "10.0.0.0/8", "64:ff9b::c0a8:101", true},
		{"ipv6 ula lifted", ProfileStrict, "fd00::/8", "fd12::1", false},
	}
	for _, c := range cases {
		p := quiet(New("t", c.profile, mustAllow(t, c.allow)))
		err := p.CheckAddr(netip.MustParseAddr(c.ip))
		if (err != nil) != c.blocked {
			t.Errorf("%s: err=%v, blocked want %v", c.name, err, c.blocked)
		}
	}
}

func TestParseAllowListAndProfile(t *testing.T) {
	got, err := ParseAllowList(" 10.0.0.0/8, 192.168.1.5;fd00::/8\n172.16.5.9/12 ")
	if err != nil || len(got) != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[3].String() != "172.16.0.0/12" {
		t.Errorf("prefix is not masked: %v", got[3])
	}
	if got, err := ParseAllowList(""); err != nil || got != nil {
		t.Errorf("empty list: %v, %v", got, err)
	}
	for _, bad := range []string{"10.0.0.0/33", "abc", "10.0.0.0/8,xx", "fe80::1%eth0"} {
		if _, err := ParseAllowList(bad); err == nil {
			t.Errorf("ParseAllowList(%q): expected an error", bad)
		}
	}
	for in, want := range map[string]Profile{"": ProfileOff, "off": ProfileOff, "Strict": ProfileStrict, " permissive ": ProfilePermissive} {
		if got, err := ParseProfile(in); err != nil || got != want {
			t.Errorf("ParseProfile(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseProfile("paranoid"); err == nil {
		t.Error("unknown profile must be an error")
	}
	if New("x", ProfileOff, nil) != nil {
		t.Error("ProfileOff must give a nil policy")
	}
}

func TestCheckURL(t *testing.T) {
	strict := quiet(New("t", ProfileStrict, nil))
	permissive := quiet(New("t", ProfilePermissive, nil))
	cases := []struct {
		url        string
		strictFail bool
		permFail   bool
		blocked    bool // ошибка именно от политики (а не синтаксис/схема)
	}{
		{"https://example.com/index.yaml", false, false, false},
		{"http://8.8.8.8:8080/x", false, false, false},
		{"http://127.0.0.1/", true, true, true},
		{"http://localhost.example/", false, false, false}, // имя: решит Control на соединении
		{"http://2130706433/", true, true, true},
		{"http://0177.0.0.1/", true, true, true},
		{"http://0x7f.1/", true, true, true},
		{"http://[::1]:8080/", true, true, true},
		{"http://[::ffff:127.0.0.1]/", true, true, true},
		{"http://169.254.169.254/latest/meta-data/", true, true, true},
		{"http://2852039166/", true, true, true},
		{"http://[fd00:ec2::254]/", true, true, true},
		{"http://[64:ff9b::a9fe:a9fe]/", true, true, true},
		{"http://user:pass@169.254.169.254/", true, true, true},
		{"http://10.0.0.1/", true, false, true},
		{"https://192.168.0.10:6443", true, false, true},
		{"http://100.64.0.1/", true, false, true},
		{"http://[fc00::1]/", true, false, true},
		{"file:///etc/passwd", true, true, false},
		{"gopher://127.0.0.1:70/_x", true, true, false},
		{"ftp://example.com/", true, true, false},
		{"javascript:alert(1)", true, true, false},
		{"//example.com/x", true, true, false},
		{"http:///path", true, true, false},
		{"http://", true, true, false},
		{"http://exa mple.com/", true, true, false},
	}
	for _, c := range cases {
		se, pe := strict.CheckURL(c.url), permissive.CheckURL(c.url)
		if (se != nil) != c.strictFail {
			t.Errorf("strict %q: err=%v, fail want %v", c.url, se, c.strictFail)
		}
		if (pe != nil) != c.permFail {
			t.Errorf("permissive %q: err=%v, fail want %v", c.url, pe, c.permFail)
		}
		if se != nil && IsBlocked(se) != c.blocked {
			t.Errorf("strict %q: IsBlocked=%v, want %v (%v)", c.url, IsBlocked(se), c.blocked, se)
		}
	}
}

func TestNilPolicyDoesNothing(t *testing.T) {
	var p *Policy
	if p.CheckURL("file:///etc/passwd") != nil || p.CheckAddr(netip.MustParseAddr("127.0.0.1")) != nil || p.Control("tcp", "127.0.0.1:1", nil) != nil {
		t.Error("nil policy must not restrict anything")
	}
	if err := p.CheckRedirect(nil, make([]*http.Request, 10)); err == nil {
		t.Error("the redirect limit must apply to a nil policy too")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	resp, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL)
	if err != nil {
		t.Fatalf("a nil policy must connect to loopback: %v", err)
	}
	resp.Body.Close()
}

// Блокировка выглядит как обычный отказ в соединении и не раскрывает причину.
func TestBlockedLooksLikeRefused(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	p := quiet(New("t", ProfilePermissive, nil))
	_, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL)
	if err == nil {
		t.Fatal("loopback must be blocked")
	}
	if !IsBlocked(err) || BlockReason(err) != ReasonLoopback {
		t.Errorf("IsBlocked=%v reason=%q (%v)", IsBlocked(err), BlockReason(err), err)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Errorf("must unwrap to ECONNREFUSED: %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "connection refused") || strings.Contains(strings.ToLower(msg), "block") || strings.Contains(strings.ToLower(msg), "policy") {
		t.Errorf("message must look like a refused connection: %q", msg)
	}
	if hits.Load() != 0 {
		t.Error("the server must not receive the request")
	}
}

func TestAllowedLoopbackConnects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()
	p := quiet(New("t", ProfileStrict, mustAllow(t, "127.0.0.0/8")))
	resp, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

// fakeResolver: первый ответ (проверка) — публичный адрес, дальше (соединение) — loopback.
type fakeResolver struct {
	mu    sync.Mutex
	calls int
	first netip.Addr
	next  netip.Addr
}

func (f *fakeResolver) LookupNetIP(_ context.Context, _, _ string) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls == 1 {
		return []netip.Addr{f.first}, nil
	}
	return []netip.Addr{f.next}, nil
}

func TestDNSRebindingIsBlocked(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())

	res := &fakeResolver{first: netip.MustParseAddr("93.184.216.34"), next: netip.MustParseAddr("127.0.0.1")}
	p := quiet(New("t", ProfileStrict, nil))
	p.Resolver = res

	// «Проверка заранее»: имя резолвится в публичный адрес и выглядит безопасным.
	pre, _ := res.LookupNetIP(context.Background(), "ip", "rebind.example")
	if err := p.CheckAddr(pre[0]); err != nil {
		t.Fatalf("pre-check must pass: %v", err)
	}
	// Соединение: DNS уже отвечает 127.0.0.1.
	_, err := p.NewHTTPClient(5 * time.Second).Get("http://rebind.example:" + port + "/")
	if err == nil || !IsBlocked(err) {
		t.Fatalf("rebinding must be blocked at connect time: %v", err)
	}
	if hits.Load() != 0 {
		t.Error("the server must not receive the request")
	}
}

func TestMixedDNSAnswerOnlyConnectsToAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	// Публичный адрес недостижим из теста, поэтому проверяем лишь, что loopback из ответа не используется.
	res := &fakeResolver{first: netip.MustParseAddr("127.0.0.1"), next: netip.MustParseAddr("169.254.169.254")}
	p := quiet(New("t", ProfilePermissive, nil))
	p.Resolver = res
	p.DialTimeout = time.Second
	if _, err := p.DialContext(context.Background(), "tcp", "mixed.example:"+port); !IsBlocked(err) {
		t.Fatalf("expected a block: %v", err)
	}
}

func TestRedirectsAreRechecked(t *testing.T) {
	var metaHits atomic.Int32
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { metaHits.Add(1) }))
	defer meta.Close()

	targets := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://2852039166/",
		"http://[fd00:ec2::254]/",
		"http://[::1]:1/",
		"file:///etc/passwd",
		"gopher://127.0.0.1:70/",
	}
	for _, target := range targets {
		target := target
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target, http.StatusFound)
		}))
		// Исходный сервер разрешён исключением, цель редиректа — нет.
		p := quiet(New("t", ProfilePermissive, mustAllow(t, "127.0.0.1/32")))
		_, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL)
		srv.Close()
		if err == nil {
			t.Errorf("redirect to %s must fail", target)
			continue
		}
		if strings.HasPrefix(target, "http") && !IsBlocked(err) {
			t.Errorf("redirect to %s: expected a block, got %v", target, err)
		}
		if strings.HasPrefix(target, "file") && !strings.Contains(err.Error(), "unsupported URL scheme") {
			t.Errorf("redirect to %s: %v", target, err)
		}
	}
}

func TestRedirectToHostnameResolvingToMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example/", http.StatusFound)
	}))
	defer srv.Close()
	p := quiet(New("t", ProfilePermissive, mustAllow(t, "127.0.0.1/32")))
	p.Resolver = resolverFunc(func(host string) []netip.Addr {
		return []netip.Addr{netip.MustParseAddr("169.254.169.254")}
	})
	// Исходный хост — IP-литерал; имя evil.example доходит до резолвера уже на редиректе.
	_, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL)
	if !IsBlocked(err) || BlockReason(err) != ReasonMetadata {
		t.Fatalf("expected a metadata block: %v", err)
	}
}

type resolverFunc func(host string) []netip.Addr

func (f resolverFunc) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return f(host), nil
}

func TestRedirectLimit(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	})
	p := quiet(New("t", ProfilePermissive, mustAllow(t, "127.0.0.0/8")))
	if _, err := p.NewHTTPClient(5 * time.Second).Get(srv.URL); err == nil || !strings.Contains(err.Error(), "10 redirects") {
		t.Errorf("expected the redirect limit: %v", err)
	}
}

func TestNonIPSocketsAreRejected(t *testing.T) {
	p := quiet(New("t", ProfilePermissive, nil))
	if err := p.Control("unix", "/var/run/docker.sock", nil); !IsBlocked(err) {
		t.Errorf("unix sockets must be rejected: %v", err)
	}
	if err := p.Control("tcp", "not-an-address", nil); !IsBlocked(err) {
		t.Errorf("an unparsable address must fail closed: %v", err)
	}
}

// В лог попадают только адрес и причина: ни URL, ни учётные данные, ни параметры запроса.
func TestLogHasNoSecrets(t *testing.T) {
	var buf bytes.Buffer
	p := New("helm", ProfileStrict, nil)
	p.Logger = slog.New(slog.NewTextHandler(&buf, nil))
	_ = p.CheckURL("http://admin:s3cr3tP4ss@127.0.0.1:8080/index.yaml?token=abc123")
	_, _ = p.NewHTTPClient(time.Second).Get("http://admin:s3cr3tP4ss@127.0.0.1:1/x?token=abc123")
	log := buf.String()
	if !strings.Contains(log, "reason=loopback") || !strings.Contains(log, "policy=helm") || !strings.Contains(log, "ip=127.0.0.1") {
		t.Fatalf("the log must hold the reason: %q", log)
	}
	for _, secret := range []string{"s3cr3tP4ss", "admin", "abc123", "token", "index.yaml"} {
		if strings.Contains(log, secret) {
			t.Errorf("the log leaks %q: %s", secret, log)
		}
	}
}

func TestApplyDisablesProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.internal:3128")
	tr := quiet(New("t", ProfileStrict, nil)).NewTransport()
	if tr.Proxy != nil {
		t.Error("the proxy must be disabled: a proxied request would be checked against the proxy address only")
	}
	if http.DefaultTransport.(*http.Transport).Proxy == nil {
		t.Error("the default transport must stay untouched")
	}
}

func TestRegistry(t *testing.T) {
	t.Cleanup(func() { Set(ScopeHelm, nil); Set(ScopeClusterAPI, nil) })
	if For(ScopeHelm) != nil {
		t.Fatal("a policy is off by default")
	}
	p := New("helm", ProfileStrict, nil)
	Set(ScopeHelm, p)
	if For(ScopeHelm) != p || For(ScopeClusterAPI) != nil {
		t.Error("scopes must be independent")
	}
	Set(ScopeHelm, nil)
	if For(ScopeHelm) != nil {
		t.Error("nil must clear the policy")
	}
}
