// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "net/netip"

// NormalizeSource приводит адрес клиента к ключу для ограничения частоты: IPv4 остаётся как есть,
// IPv6 сворачивается до префикса /64 (одна домашняя или офисная сеть получает тысячи адресов, и
// ограничение по полному адресу обходится сменой младших бит), IPv4-mapped IPv6 (::ffff:a.b.c.d)
// считается обычным IPv4. Строка, не являющаяся IP-адресом, возвращается без изменений.
func NormalizeSource(source string) string {
	addr, err := netip.ParseAddr(source)
	if err != nil {
		return source
	}
	addr = addr.WithZone("").Unmap()
	if addr.Is6() {
		p, err := addr.Prefix(64)
		if err != nil {
			return source
		}
		return p.String()
	}
	return addr.String()
}
