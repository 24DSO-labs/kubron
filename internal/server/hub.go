// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"log"
	"sync"
)

const (
	// connBuffer — сколько готовых кадров ждёт отправки одному клиенту. Переполнение означает
	// медленного потребителя: соединение закрывается, клиент переподключается и перечитывает состояние.
	connBuffer = 256
	// maxSubscriptions — предел подписок одного соединения (имена событий содержат идентификаторы
	// терминалов и потоков логов, поэтому набор динамический).
	maxSubscriptions = 1024
)

// Hub доставляет события ядра подписанным WebSocket-соединениям. Реализует приёмник
// webplatform.Platform.SetEmitter.
//
// Emit доставляет событие всем подписанным соединениям (общие события); EmitTo — только соединениям
// одной сессии. События сессионных видов ядра адресуются владельцу через EmitTo.
type Hub struct {
	mu    sync.RWMutex
	conns map[*Conn]struct{}
}

func NewHub() *Hub { return &Hub{conns: map[*Conn]struct{}{}} }

// eventFrame — кадр сервер → клиент: {"event": "...", "data": [...]}.
type eventFrame struct {
	Event string `json:"event"`
	Data  []any  `json:"data"`
}

// Emit рассылает событие подписчикам. Кадр сериализуется один раз; отправка не блокируется.
func (h *Hub) Emit(event string, data ...any) { h.deliver("", false, event, data) }

// EmitTo доставляет событие только соединениям сессии session (и только подписанным на него).
func (h *Hub) EmitTo(session, event string, data ...any) { h.deliver(session, true, event, data) }

func (h *Hub) deliver(session string, addressed bool, event string, data []any) {
	h.mu.RLock()
	if len(h.conns) == 0 {
		h.mu.RUnlock()
		return
	}
	var frame []byte
	for c := range h.conns {
		if addressed && c.session != session {
			continue
		}
		if !c.subscribed(event) {
			continue
		}
		if frame == nil {
			var err error
			if data == nil {
				data = []any{}
			}
			if frame, err = json.Marshal(eventFrame{Event: event, Data: data}); err != nil {
				h.mu.RUnlock()
				log.Printf("hub: событие %s не сериализуется: %v", event, err)
				return
			}
		}
		c.enqueue(frame)
	}
	h.mu.RUnlock()
}

// Register добавляет соединение сессии session.
func (h *Hub) Register(session string) *Conn {
	c := &Conn{hub: h, session: session, out: make(chan []byte, connBuffer),
		done: make(chan struct{}), subs: map[string]struct{}{}}
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// CloseSession закрывает все соединения сессии (выход, истечение, вытеснение).
func (h *Hub) CloseSession(session string) {
	h.mu.RLock()
	var victims []*Conn
	for c := range h.conns {
		if c.session == session {
			victims = append(victims, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range victims {
		c.Close()
	}
}

// Len — число соединений.
func (h *Hub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// Conn — одно соединение клиента.
type Conn struct {
	hub     *Hub
	session string

	out  chan []byte
	done chan struct{}

	mu     sync.Mutex
	subs   map[string]struct{}
	closed bool
}

// Out — готовые кадры для записи в сокет.
func (c *Conn) Out() <-chan []byte { return c.out }

// Done закрывается при закрытии соединения.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Session — идентификатор веб-сессии владельца.
func (c *Conn) Session() string { return c.session }

func (c *Conn) subscribed(event string) bool {
	c.mu.Lock()
	_, ok := c.subs[event]
	c.mu.Unlock()
	return ok
}

// Subscribe добавляет подписки; при превышении предела подписывает на часть и возвращает false.
func (c *Conn) Subscribe(events []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range events {
		if _, ok := c.subs[e]; ok {
			continue
		}
		if len(c.subs) >= maxSubscriptions {
			return false
		}
		c.subs[e] = struct{}{}
	}
	return true
}

func (c *Conn) Unsubscribe(events []string) {
	c.mu.Lock()
	for _, e := range events {
		delete(c.subs, e)
	}
	c.mu.Unlock()
}

// Send ставит служебный кадр (подтверждение, pong) в очередь; false — очередь полна, соединение закрыто.
func (c *Conn) Send(frame []byte) bool { return c.enqueue(frame) }

func (c *Conn) enqueue(frame []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.out <- frame:
		return true
	default:
		go c.Close() // медленный потребитель; из-под RLock хаба закрываем асинхронно
		return false
	}
}

// Close закрывает соединение и убирает его из хаба (идемпотентно).
func (c *Conn) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	c.mu.Unlock()
	c.hub.mu.Lock()
	delete(c.hub.conns, c)
	c.hub.mu.Unlock()
}

// clientMsg — кадр клиент → сервер.
type clientMsg struct {
	Op     string   `json:"op"` // sub | unsub | ping
	Events []string `json:"events"`
}

// Handle применяет кадр клиента и возвращает ответный кадр (nil — отвечать нечем).
// Подтверждение подписки нужно клиенту, чтобы начинать поток (терминал, логи) уже после
// подписки и не терять первые данные.
func (c *Conn) Handle(raw []byte) []byte {
	var m clientMsg
	if err := json.Unmarshal(raw, &m); err != nil {
		return mustJSON(map[string]any{"op": "error", "message": "ожидается JSON"})
	}
	switch m.Op {
	case "sub":
		if !c.Subscribe(m.Events) {
			return mustJSON(map[string]any{"op": "error", "message": "слишком много подписок"})
		}
		return mustJSON(map[string]any{"op": "sub_ok", "events": m.Events})
	case "unsub":
		c.Unsubscribe(m.Events)
		return mustJSON(map[string]any{"op": "unsub_ok", "events": m.Events})
	case "ping":
		return mustJSON(map[string]any{"op": "pong"})
	default:
		return mustJSON(map[string]any{"op": "error", "message": "неизвестная операция"})
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
