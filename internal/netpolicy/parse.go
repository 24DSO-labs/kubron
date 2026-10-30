// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package netpolicy

import (
	"fmt"
	"net/netip"
	"strings"
	"unicode"
)

// ParseIPHost распознаёт host как IP-адрес в любой записи, которую понимают сетевые библиотеки:
// обычный IPv4/IPv6 (в том числе в скобках, с зоной и ::ffff:a.b.c.d) и «нестандартный» IPv4 в духе
// inet_aton — десятичный целым числом (2130706433), восьмеричный (0177.0.0.1), шестнадцатеричный
// (0x7f.1), с 1–3 частями (127.1). Не IP (имя хоста) — ok == false.
// Адрес возвращается нормализованным: без зоны и без обёртки IPv4-mapped.
func ParseIPHost(host string) (netip.Addr, bool) {
	h := host
	if strings.HasPrefix(h, "[") && strings.HasSuffix(h, "]") {
		h = h[1 : len(h)-1]
	}
	h = strings.TrimSuffix(h, ".") // «127.0.0.1.» — тот же адрес
	if h == "" {
		return netip.Addr{}, false
	}
	if a, err := netip.ParseAddr(h); err == nil {
		return a.WithZone("").Unmap(), true
	}
	if strings.ContainsAny(h, ":%") {
		return netip.Addr{}, false
	}
	if a, ok := parseLegacyIPv4(h); ok {
		return a, true
	}
	return netip.Addr{}, false
}

// parseLegacyIPv4 — правила inet_aton: 1–4 части, каждая в десятичной, восьмеричной (ведущий 0) или
// шестнадцатеричной (0x) записи; последняя часть занимает все оставшиеся байты.
func parseLegacyIPv4(s string) (netip.Addr, bool) {
	parts := strings.Split(s, ".")
	if len(parts) > 4 {
		return netip.Addr{}, false
	}
	vals := make([]uint64, len(parts))
	for i, p := range parts {
		v, ok := parseIPNumber(p)
		if !ok {
			return netip.Addr{}, false
		}
		vals[i] = v
	}
	var out uint64
	for i := 0; i < len(vals)-1; i++ {
		if vals[i] > 255 {
			return netip.Addr{}, false
		}
		out |= vals[i] << (8 * uint(3-i))
	}
	last := vals[len(vals)-1]
	if last >= uint64(1)<<(8*uint(4-(len(vals)-1))) {
		return netip.Addr{}, false
	}
	out |= last
	return netip.AddrFrom4([4]byte{byte(out >> 24), byte(out >> 16), byte(out >> 8), byte(out)}), true
}

func parseIPNumber(s string) (uint64, bool) {
	base := uint64(10)
	switch {
	case len(s) > 1 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X'):
		base, s = 16, s[2:]
	case len(s) > 1 && s[0] == '0':
		base, s = 8, s[1:]
	}
	if s == "" {
		return 0, false
	}
	var v uint64
	for _, c := range s {
		var d uint64
		switch {
		case c >= '0' && c <= '9':
			d = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint64(c-'A') + 10
		default:
			return 0, false
		}
		if d >= base {
			return 0, false
		}
		v = v*base + d
		if v > 0xFFFFFFFF {
			return 0, false
		}
	}
	return v, true
}

// ParseAllowList разбирает список исключений из переменной окружения: CIDR или одиночные адреса через
// запятую и/или пробелы. Пустая строка — пустой список.
func ParseAllowList(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		var (
			p   netip.Prefix
			err error
		)
		if strings.Contains(f, "/") {
			p, err = netip.ParsePrefix(f)
		} else {
			var a netip.Addr
			if a, err = netip.ParseAddr(f); err == nil {
				p = netip.PrefixFrom(a, a.BitLen())
				if a.Zone() != "" { // PrefixFrom зону отбрасывает: проверяем сам адрес
					return nil, fmt.Errorf("invalid CIDR or address %q in the allow list", f)
				}
			}
		}
		if err != nil || p.Addr().Zone() != "" {
			return nil, fmt.Errorf("invalid CIDR or address %q in the allow list", f)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

// ParseProfile разбирает значение переменной окружения: "", "off", "permissive" или "strict".
func ParseProfile(s string) (Profile, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return ProfileOff, nil
	case "permissive":
		return ProfilePermissive, nil
	case "strict":
		return ProfileStrict, nil
	}
	return ProfileOff, fmt.Errorf("unknown network policy profile %q", s)
}
