// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func reqFrom(remote string, xff ...string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/rpc/Login", nil)
	r.RemoteAddr = remote
	for _, v := range xff {
		r.Header.Add("X-Forwarded-For", v)
	}
	return r
}

func mustCIDRs(t *testing.T, list string) []netip.Prefix {
	t.Helper()
	p, err := ParseTrustedProxies(list)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestClientIPWithoutTrustIgnoresForwardedFor(t *testing.T) {
	p := ProxyTrust{}
	if got := p.ClientIP(reqFrom("10.1.2.3:5555", "6.6.6.6, 203.0.113.9")); got != "10.1.2.3" {
		t.Errorf("XFF must be ignored without trust: %q", got)
	}
}

// Клиент подделывает левую часть X-Forwarded-For; прокси дописывает настоящий адрес справа.
func TestClientIPTakesTheRightmostNotTheForgedLeftElement(t *testing.T) {
	p := ProxyTrust{Enabled: true, Hops: 1}
	for name, c := range map[string]struct {
		remote string
		xff    []string
		want   string
	}{
		"single proxy, no forgery":       {"10.0.0.1:1", []string{"203.0.113.9"}, "203.0.113.9"},
		"client forged the left element": {"10.0.0.1:1", []string{"1.2.3.4, 203.0.113.9"}, "203.0.113.9"},
		"client forged many":             {"10.0.0.1:1", []string{"1.1.1.1, 2.2.2.2, 3.3.3.3, 203.0.113.9"}, "203.0.113.9"},
		"forged in a separate header":    {"10.0.0.1:1", []string{"6.6.6.6", "203.0.113.9"}, "203.0.113.9"},
		"no header at all":               {"10.0.0.1:1", nil, "10.0.0.1"},
		"garbage is not an address":      {"10.0.0.1:1", []string{"unknown"}, "10.0.0.1"},
		"port is stripped":               {"10.0.0.1:1", []string{"203.0.113.9:4444"}, "203.0.113.9"},
		"IPv6 with port in brackets":     {"10.0.0.1:1", []string{"[2001:db8::7]:443"}, "2001:db8::7"},
		"IPv4-mapped is unwrapped":       {"10.0.0.1:1", []string{"::ffff:203.0.113.9"}, "203.0.113.9"},
		"empty elements are skipped":     {"10.0.0.1:1", []string{" , ,203.0.113.9, "}, "203.0.113.9"},
	} {
		if got := p.ClientIP(reqFrom(c.remote, c.xff...)); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestClientIPHops(t *testing.T) {
	p := ProxyTrust{Enabled: true, Hops: 2}
	for name, c := range map[string]struct {
		xff  string
		want string
	}{
		// клиент → балансировщик → ingress → сервер: XFF = «клиент, балансировщик»
		"two proxies":              {"203.0.113.9, 10.0.0.2", "203.0.113.9"},
		"forged prefix is ignored": {"6.6.6.6, 203.0.113.9, 10.0.0.2", "203.0.113.9"},
		// цепочка короче ожидаемой: запрос пришёл в обход балансировщика, верить нельзя
		"chain shorter than hops": {"6.6.6.6", "192.0.2.77"},
	} {
		if got := p.ClientIP(reqFrom("192.0.2.77:1", c.xff)); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
	// Hops не задан (0) — как 1
	if got := (ProxyTrust{Enabled: true}).ClientIP(reqFrom("10.0.0.1:1", "6.6.6.6, 203.0.113.9")); got != "203.0.113.9" {
		t.Errorf("default hops: %q", got)
	}
}

func TestClientIPWithTrustedCIDRs(t *testing.T) {
	p := ProxyTrust{Enabled: true, CIDRs: mustCIDRs(t, "10.0.0.0/8, 192.168.1.5, fd00::/8")}
	for name, c := range map[string]struct {
		remote string
		xff    []string
		want   string
	}{
		"proxy in the list":                {"10.0.0.1:1", []string{"203.0.113.9"}, "203.0.113.9"},
		"forged trusted-looking left part": {"10.0.0.1:1", []string{"10.9.9.9, 198.51.100.8"}, "198.51.100.8"},
		"two proxies in the list":          {"10.0.0.1:1", []string{"203.0.113.9, 10.0.0.7"}, "203.0.113.9"},
		"forged beyond the real client":    {"10.0.0.1:1", []string{"6.6.6.6, 203.0.113.9, 10.0.0.7"}, "203.0.113.9"},
		"single address entry":             {"192.168.1.5:1", []string{"203.0.113.9"}, "203.0.113.9"},
		"IPv6 proxy":                       {"[fd00::1]:1", []string{"203.0.113.9"}, "203.0.113.9"},
		"peer is NOT a trusted proxy":      {"192.0.2.1:1", []string{"6.6.6.6"}, "192.0.2.1"},
		"all hops trusted: leftmost":       {"10.0.0.1:1", []string{"10.2.2.2, 10.0.0.7"}, "10.2.2.2"},
		"garbage in the chain":             {"10.0.0.1:1", []string{"203.0.113.9, nonsense"}, "10.0.0.1"},
		"no header":                        {"10.0.0.1:1", nil, "10.0.0.1"},
	} {
		if got := p.ClientIP(reqFrom(c.remote, c.xff...)); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestParseTrustedProxies(t *testing.T) {
	got, err := ParseTrustedProxies(" 10.1.2.3/8 , 192.168.0.1,, ::ffff:172.16.0.9 ")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/8", "192.168.0.1/32", "172.16.0.9/32"}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Errorf("%d: %s, want %s", i, got[i], want[i])
		}
	}
	if p, err := ParseTrustedProxies(""); err != nil || len(p) != 0 {
		t.Errorf("empty list: %v %v", p, err)
	}
	for _, bad := range []string{"not-an-ip", "10.0.0.0/33", "10.0.0.1-10.0.0.9"} {
		if _, err := ParseTrustedProxies(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}
