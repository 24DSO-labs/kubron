// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ProxyTrust описывает, каким обратным прокси верить при определении адреса клиента по X-Forwarded-For.
//
// Каждый прокси дописывает в конец заголовка адрес, от которого получил запрос, поэтому левые элементы
// клиент может подделать, а правые добавлены нашей инфраструктурой. Адрес клиента берётся справа,
// после последнего доверенного прокси.
type ProxyTrust struct {
	// Enabled — главный выключатель (KUBRON_TRUST_PROXY). Выключен: X-Forwarded-For игнорируется целиком.
	Enabled bool
	// Hops — сколько доверенных прокси стоит перед сервером, считая прямого собеседника (по умолчанию 1).
	// Клиент — элемент X-Forwarded-For с индексом len-Hops. Если элементов меньше, цепочка короче
	// ожидаемой (в обход прокси) и берётся адрес прямого собеседника. Не используется, если задан CIDRs.
	Hops int
	// CIDRs — адреса доверенных прокси (KUBRON_TRUSTED_PROXIES). Если задан, прямой собеседник должен
	// входить в список (иначе заголовок игнорируется), а клиент — первый справа адрес вне списка.
	CIDRs []netip.Prefix
}

// ClientIP возвращает адрес клиента запроса. При любой неоднозначности (мусор в заголовке, цепочка
// короче ожидаемой, недоверенный собеседник) возвращается адрес прямого собеседника: подделать его нельзя.
func (p ProxyTrust) ClientIP(r *http.Request) string {
	remote := remoteHost(r)
	if !p.Enabled {
		return remote
	}
	var chain []string
	for _, line := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(line, ",") {
			if part = strings.TrimSpace(part); part != "" {
				chain = append(chain, part)
			}
		}
	}
	if len(chain) == 0 {
		return remote
	}

	if len(p.CIDRs) > 0 {
		peer, err := netip.ParseAddr(remote)
		if err != nil || !p.trusted(peer) {
			return remote // заголовок прислал не наш прокси
		}
		for i := len(chain) - 1; i >= 0; i-- {
			a, ok := parseForwarded(chain[i])
			if !ok {
				return remote
			}
			if !p.trusted(a) {
				return a.String()
			}
		}
		// вся цепочка — доверенные прокси: клиент внутри доверенной сети, берём самый левый адрес
		if a, ok := parseForwarded(chain[0]); ok {
			return a.String()
		}
		return remote
	}

	hops := max(p.Hops, 1)
	if len(chain) < hops {
		return remote
	}
	if a, ok := parseForwarded(chain[len(chain)-hops]); ok {
		return a.String()
	}
	return remote
}

func (p ProxyTrust) trusted(a netip.Addr) bool {
	a = a.Unmap()
	for _, c := range p.CIDRs {
		if c.Contains(a) {
			return true
		}
	}
	return false
}

// parseForwarded разбирает элемент X-Forwarded-For: «203.0.113.9», «[2001:db8::1]:443», «2001:db8::1».
func parseForwarded(s string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.WithZone("").Unmap(), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().WithZone("").Unmap(), true
	}
	return netip.Addr{}, false
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ParseTrustedProxies разбирает список доверенных прокси «10.0.0.0/8, 192.168.1.5, fd00::/8»:
// CIDR или одиночные адреса через запятую.
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, e := range strings.Split(list, ",") {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if p, err := netip.ParsePrefix(e); err == nil {
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(e)
		if err != nil {
			return nil, err
		}
		a = a.WithZone("").Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}
