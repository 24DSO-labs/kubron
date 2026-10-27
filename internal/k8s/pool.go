// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"sync"
)

// connPool — общий для всех сессий пул подключений к кластерам.
//
// Подключение (клиенты client-go, informers) создаётся один раз и используется всеми
// «видами» Manager (см. Manager.ForSession). refs — сколько видов сейчас держат кластер
// активным; когда счётчик падает до нуля, фоновые informers останавливаются (сам клиент
// остаётся в пуле и переиспользуется при следующей активации).
//
// СПАЙК: создание подключения идёт под блокировкой пула; для вызовов, которые ходят в сеть
// (нативный oidc-password), нужен singleflight по ключу — см. ADR-0005.
type connPool struct {
	mu      sync.Mutex
	entries map[string]*poolEntry
	// closeFn вызывается, когда у кластера не осталось активных видов (по умолчанию — Close).
	closeFn func(*ClusterConnection)
}

type poolEntry struct {
	conn *ClusterConnection
	refs int
	// holders — виды, державшие кластер активным: им адресуются события кластера.
	holders map[*Manager]struct{}
}

func newConnPool() *connPool {
	return &connPool{
		entries: make(map[string]*poolEntry),
		closeFn: func(c *ClusterConnection) { c.Close() },
	}
}

func (p *connPool) get(id string) (*ClusterConnection, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.entries[id]
	if !ok {
		return nil, false
	}
	return e.conn, true
}

// put кладёт готовое подключение (если уже есть — возвращает существующее).
func (p *connPool) put(id string, conn *ClusterConnection) *ClusterConnection {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.entries[id]; ok {
		return e.conn
	}
	p.entries[id] = &poolEntry{conn: conn}
	return conn
}

// getOrCreate возвращает существующее подключение или создаёт его через build.
// build выполняется под блокировкой пула, поэтому параллельные вызовы для одного
// кластера не создают два подключения.
func (p *connPool) getOrCreate(id string, build func() (*ClusterConnection, error)) (*ClusterConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.entries[id]; ok {
		return e.conn, nil
	}
	conn, err := build()
	if err != nil {
		return nil, err
	}
	p.entries[id] = &poolEntry{conn: conn}
	return conn, nil
}

func (p *connPool) has(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.entries[id]
	return ok
}

// acquire отмечает, что ещё один вид держит кластер активным.
func (p *connPool) acquire(id string, holder *Manager) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.entries[id]
	if !ok {
		return fmt.Errorf("кластер %s не подключён", id)
	}
	e.refs++
	if holder != nil {
		if e.holders == nil {
			e.holders = map[*Manager]struct{}{}
		}
		e.holders[holder] = struct{}{}
	}
	return nil
}

// holdersOf — виды, у которых кластер сейчас активен.
func (p *connPool) holdersOf(id string) []*Manager {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.entries[id]
	if !ok {
		return nil
	}
	out := make([]*Manager, 0, len(e.holders))
	for h := range e.holders {
		out = append(out, h)
	}
	return out
}

// release снимает отметку; при нуле останавливает informers кластера.
func (p *connPool) release(id string, holder *Manager) {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.entries[id]
	if !ok {
		return
	}
	delete(e.holders, holder)
	if e.refs > 0 {
		e.refs--
	}
	if e.refs == 0 {
		p.closeFn(e.conn)
	}
}

// remove закрывает и забывает подключение. Виды, державшие его активным, при следующем
// GetActiveConnection получат ошибку «нет активного подключения».
func (p *connPool) remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.entries[id]; ok {
		p.closeFn(e.conn)
		delete(p.entries, id)
	}
}
