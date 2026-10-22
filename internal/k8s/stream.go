package k8s

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Имена событий Wails; к префиксу добавляется id потока / сессии.
const (
	EventTerminalData = "terminal:data:" // payload: base64 сырых байт
	EventTerminalExit = "terminal:exit:"
	EventLogsData     = "logs:data:" // payload: строка, только целые строки
	EventLogsEnd      = "logs:end:"  // payload: причина ("eof" или текст ошибки)

	logFlushEvery  = 100 * time.Millisecond
	logMaxPending  = 256 << 10
	logMaxStreams  = 50
	logConnectWait = 20 * time.Second
)

// ─── Терминал ───

// pumpTerminal читает вывод сессии и шлёт его фронтенду событиями (вместо опроса ReadTerminal).
// Завершается, когда поток закрыт: по EOF (процесс вышел) или после CloseTerminal.
func (m *Manager) pumpTerminal(s *TerminalSession) {
	buf := make([]byte, 32<<10)
	for {
		n, err := s.stdoutR.Read(buf)
		if n > 0 {
			m.emit(EventTerminalData+s.ID, base64.StdEncoding.EncodeToString(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	m.emit(EventTerminalExit + s.ID)
}

// ─── Логи ───

type logEntry struct{ cancel context.CancelFunc }

type logStreams struct {
	mu sync.Mutex
	m  map[string]*logEntry
}

func (l *logStreams) add(id string, e *logEntry) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.m == nil {
		l.m = map[string]*logEntry{}
	}
	if old, ok := l.m[id]; ok {
		old.cancel()
	} else if len(l.m) >= logMaxStreams {
		return false
	}
	l.m[id] = e
	return true
}

// remove удаляет запись, только если под этим id не успели зарегистрировать новую.
func (l *logStreams) remove(id string, e *logEntry) {
	l.mu.Lock()
	if l.m[id] == e {
		delete(l.m, id)
	}
	l.mu.Unlock()
}

func (l *logStreams) stop(id string) {
	l.mu.Lock()
	if e, ok := l.m[id]; ok {
		e.cancel()
		delete(l.m, id)
	}
	l.mu.Unlock()
}

func (l *logStreams) stopAll() {
	l.mu.Lock()
	for id, e := range l.m {
		e.cancel()
		delete(l.m, id)
	}
	l.mu.Unlock()
}

// StartLogStream запускает follow-поток логов контейнера. Данные уходят событиями
// logs:data:<id> (пачками раз в 100 мс, только целые строки), конец — logs:end:<id>.
// Фронтенд должен подписаться на события ДО вызова.
func (m *Manager) StartLogStream(streamID, namespace, pod, container string, tail int64) error {
	if streamID == "" {
		return errors.New("не указан id потока логов")
	}
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	if tail <= 0 || tail > 10000 {
		tail = 100
	}
	opts := &corev1.PodLogOptions{Container: container, TailLines: &tail, Timestamps: true, Follow: true}

	ctx, cancel := context.WithCancel(context.Background())
	connectTimer := time.AfterFunc(logConnectWait, cancel) // подключение не должно висеть вечно
	stream, err := conn.Client.CoreV1().Pods(namespace).GetLogs(pod, opts).Stream(ctx)
	connectTimer.Stop()
	if err != nil {
		cancel()
		return fmt.Errorf("не удалось получить логи: %w", err)
	}
	entry := &logEntry{cancel: cancel}
	if !m.logs.add(streamID, entry) {
		cancel()
		stream.Close()
		return fmt.Errorf("слишком много открытых потоков логов (максимум %d)", logMaxStreams)
	}
	go m.pumpLogs(ctx, streamID, entry, stream)
	return nil
}

func (m *Manager) StopLogStream(streamID string) { m.logs.stop(streamID) }
func (m *Manager) StopAllLogStreams()            { m.logs.stopAll() }

// splitComplete делит буфер на целые строки и неполный хвост: так UTF-8 и строки не рвутся между событиями.
func splitComplete(b []byte) (send, rest []byte) {
	i := bytes.LastIndexByte(b, '\n')
	if i < 0 {
		return nil, b
	}
	return b[:i+1], b[i+1:]
}

func (m *Manager) pumpLogs(ctx context.Context, id string, entry *logEntry, stream io.ReadCloser) {
	defer m.logs.remove(id, entry)
	defer entry.cancel()
	defer stream.Close()

	chunks := make(chan []byte, 16)
	var readErr error
	go func() {
		defer close(chunks)
		buf := make([]byte, 32<<10)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				select {
				case chunks <- append([]byte(nil), buf[:n]...):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				readErr = err // читается только после close(chunks)
				return
			}
		}
	}()

	var pending []byte
	flush := func(force bool) {
		if len(pending) == 0 {
			return
		}
		send, rest := splitComplete(pending)
		if force || len(pending) > logMaxPending {
			send, rest = pending, nil
		}
		if len(send) > 0 {
			m.emit(EventLogsData+id, string(send))
		}
		pending = append(pending[:0], rest...)
	}

	tick := time.NewTicker(logFlushEvery)
	defer tick.Stop()
	for {
		select {
		case c, ok := <-chunks:
			if !ok {
				flush(true)
				reason := "eof"
				if readErr != nil && !errors.Is(readErr, io.EOF) && ctx.Err() == nil {
					reason = readErr.Error()
				}
				m.emit(EventLogsEnd+id, reason)
				return
			}
			pending = append(pending, c...)
		case <-tick.C:
			flush(false)
		case <-ctx.Done():
			return
		}
	}
}
