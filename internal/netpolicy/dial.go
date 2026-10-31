// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package netpolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultDialTimeout = 10 * time.Second
	maxRedirects       = 10
)

// Control подходит для net.Dialer.Control: вызывается для каждого соединения с уже разрешённым
// числовым адресом, поэтому подмена DNS между проверкой и соединением ничего не даёт.
func (p *Policy) Control(network, address string, _ syscall.RawConn) error {
	if p == nil {
		return nil
	}
	if !strings.HasPrefix(network, "tcp") && !strings.HasPrefix(network, "udp") {
		return p.deny("dial", netip.Addr{}, ReasonInvalid) // сокеты, отличные от IP, политика не обслуживает
	}
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return p.deny("dial", netip.Addr{}, ReasonInvalid)
	}
	if r := p.verdict(ap.Addr()); r != "" {
		return p.deny("dial", ap.Addr(), r)
	}
	return nil
}

func (p *Policy) resolver() Resolver {
	if p.Resolver != nil {
		return p.Resolver
	}
	return net.DefaultResolver
}

func (p *Policy) dialTimeout() time.Duration {
	if p.DialTimeout > 0 {
		return p.DialTimeout
	}
	return defaultDialTimeout
}

// ipNetwork переводит сеть сокета ("tcp4") в сеть для LookupNetIP ("ip4").
func ipNetwork(network string) string {
	switch {
	case strings.HasSuffix(network, "4"):
		return "ip4"
	case strings.HasSuffix(network, "6"):
		return "ip6"
	}
	return "ip"
}

// DialContext подходит для http.Transport.DialContext и rest.Config.Dial. Имя разрешается здесь, адреса
// перебираются по одному, а соединение открывается по числовому адресу под защитой Control. Записи IP в
// нестандартном виде (2130706433, 0x7f.1) приводятся к обычным и не доходят до системного резолвера.
func (p *Policy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if p == nil {
		return d.DialContext(ctx, network, address)
	}
	d.Timeout = p.dialTimeout()
	d.Control = p.Control

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var addrs []netip.Addr
	if ip, ok := ParseIPHost(host); ok {
		addrs = []netip.Addr{ip}
	} else {
		found, err := p.resolver().LookupNetIP(ctx, ipNetwork(network), host)
		if err != nil {
			return nil, err
		}
		for _, a := range found {
			addrs = append(addrs, a.WithZone("").Unmap())
		}
	}
	if len(addrs) == 0 {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}

	var lastErr error
	for _, a := range addrs {
		if (network == "tcp4" && !a.Is4()) || (network == "tcp6" && !a.Is6()) {
			continue
		}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(a.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	if lastErr == nil {
		lastErr = &net.DNSError{Err: "no suitable address found", Name: host}
	}
	return nil, lastErr
}

// Apply направляет все соединения транспорта через политику. Прокси из окружения отключается:
// иначе запрос к внутреннему адресу ушёл бы на прокси, а проверялся бы только адрес прокси.
func (p *Policy) Apply(tr *http.Transport) *http.Transport {
	if p == nil {
		return tr
	}
	tr.DialContext = p.DialContext
	tr.DialTLSContext = nil
	tr.Dial = nil    //nolint:staticcheck // устаревшее поле не должно обходить проверку
	tr.DialTLS = nil //nolint:staticcheck
	tr.Proxy = nil
	return tr
}

// NewTransport — копия http.DefaultTransport под политикой (для nil-политики — просто копия).
func (p *Policy) NewTransport() *http.Transport {
	return p.Apply(http.DefaultTransport.(*http.Transport).Clone())
}

// NewHTTPClient — клиент, у которого и соединения, и редиректы проходят через политику.
func (p *Policy) NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: p.NewTransport(), Timeout: timeout, CheckRedirect: p.CheckRedirect}
}

// CheckURL — быстрая проверка URL до запроса: схема (только http и https: file://, gopher://, ftp:// и
// прочие отклоняются) и адрес, если хост задан IP-литералом в любой записи. Имена не разрешаются: DNS
// отвечает по-разному от запроса к запросу, и проверка имени заранее ничего не гарантирует; решает Control.
func (p *Policy) CheckURL(raw string) error {
	if p == nil {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid URL") // текст url.Parse содержит сам URL, а в нём могут быть учётные данные
	}
	return p.checkParsed(u, "url")
}

func (p *Policy) checkParsed(u *url.URL, stage string) error {
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("URL has no host")
	}
	ip, ok := ParseIPHost(host)
	if !ok {
		return nil
	}
	if r := p.verdict(ip); r != "" {
		port := uint16(80)
		if scheme == "https" {
			port = 443
		}
		if n, err := strconv.ParseUint(u.Port(), 10, 16); err == nil {
			port = uint16(n)
		}
		return opError(ip, port, p.deny(stage, ip, r))
	}
	return nil
}

// CheckRedirect — http.Client.CheckRedirect: каждый переход проверяется заново теми же правилами.
// Для nil-политики сохраняется только обычный предел числа переходов.
func (p *Policy) CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if p == nil {
		return nil
	}
	return p.checkParsed(req.URL, "redirect")
}
