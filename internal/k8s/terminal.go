package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// ─── Типы ──────────────────────────────────────────────────

type TerminalSession struct {
	ID        string
	Namespace string
	Pod       string
	Container string
	Shell     string

	stdinW   io.WriteCloser
	stdoutR  io.ReadCloser
	sizeChan chan remotecommand.TerminalSize
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc

	mu     sync.Mutex
	closed bool
}

type TerminalSessionManager struct {
	sessions map[string]*TerminalSession
	mu       sync.RWMutex
}

func NewTerminalSessionManager() *TerminalSessionManager {
	return &TerminalSessionManager{
		sessions: make(map[string]*TerminalSession),
	}
}

func (tsm *TerminalSessionManager) Add(session *TerminalSession) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	tsm.sessions[session.ID] = session
}

func (tsm *TerminalSessionManager) Get(sessionID string) *TerminalSession {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.sessions[sessionID]
}

func (tsm *TerminalSessionManager) Remove(sessionID string) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	delete(tsm.sessions, sessionID)
}

// ─── sizeQueue для TerminalSizeQueue ───────────────────────

type sizeQueue struct {
	sizeChan chan remotecommand.TerminalSize
	done     chan struct{}
}

func (sq *sizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size := <-sq.sizeChan:
		return &size
	case <-sq.done:
		return nil
	}
}

// ─── StartTerminalSession ──────────────────────────────────

func (m *Manager) StartTerminalSession(sessionID, namespace, pod, container, shell string, cols, rows uint16) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	// Пустой shell — режим attach к основному процессу контейнера.
	attach := shell == ""
	req := conn.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace)
	if attach {
		req = req.SubResource("attach").
			VersionedParams(&corev1.PodAttachOptions{
				Container: container,
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			}, scheme.ParameterCodec)
	} else {
		req = req.SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Container: container,
				Command:   []string{shell},
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			}, scheme.ParameterCodec)
	}

	executor, err := remotecommand.NewSPDYExecutor(conn.Config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	sizeChan := make(chan remotecommand.TerminalSize, 1)
	done := make(chan struct{})

	sizeChan <- remotecommand.TerminalSize{
		Width:  cols,
		Height: rows,
	}

	sq := &sizeQueue{
		sizeChan: sizeChan,
		done:     done,
	}

	shellLabel := shell
	if attach {
		shellLabel = "attach"
	}
	session := &TerminalSession{
		ID:        sessionID,
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		Shell:     shellLabel,
		stdinW:    stdinW,
		stdoutR:   stdoutR,
		sizeChan:  sizeChan,
		done:      done,
		ctx:       ctx,
		cancel:    cancel,
		closed:    false,
	}

	if m.terminalSessions == nil {
		m.terminalSessions = NewTerminalSessionManager()
	}
	m.terminalSessions.Add(session)

	go func() {
		executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinR,
			Stdout:            stdoutW,
			Stderr:            stdoutW,
			Tty:               true,
			TerminalSizeQueue: sq,
		})

		stdoutW.Close()
		stdinR.Close()

		if m.terminalSessions != nil {
			m.terminalSessions.Remove(sessionID)
		}

		select {
		case <-done:
		default:
			close(done)
		}
	}()

	return nil
}

// ─── WriteTerminal ─────────────────────────────────────────

func (m *Manager) WriteTerminal(sessionID string, data []byte) error {
	if m.terminalSessions == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := m.terminalSessions.Get(sessionID)
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return fmt.Errorf("session closed: %s", sessionID)
	}

	_, err := session.stdinW.Write(data)
	return err
}

// ─── ReadTerminalBytes ─────────────────────────────────────

func (m *Manager) ReadTerminalBytes(sessionID string) ([]byte, error) {
	if m.terminalSessions == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session := m.terminalSessions.Get(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	buf := make([]byte, 4096)
	n, err := session.stdoutR.Read(buf)

	if n > 0 {
		return buf[:n], nil
	}

	if err == io.EOF {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	return nil, nil
}

// ─── ResizeTerminal ────────────────────────────────────────

func (m *Manager) ResizeTerminal(sessionID string, cols, rows uint16) error {
	if m.terminalSessions == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := m.terminalSessions.Get(sessionID)
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return nil
	}

	select {
	case <-session.sizeChan:
	default:
	}

	select {
	case session.sizeChan <- remotecommand.TerminalSize{Width: cols, Height: rows}:
	default:
	}

	return nil
}

// ─── CloseTerminal ─────────────────────────────────────────

func (m *Manager) CloseTerminal(sessionID string) error {
	if m.terminalSessions == nil {
		return nil
	}

	session := m.terminalSessions.Get(sessionID)
	if session == nil {
		return nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return nil
	}

	session.closed = true
	session.cancel()
	session.stdinW.Close()
	session.stdoutR.Close()

	select {
	case <-session.done:
	default:
		close(session.done)
	}

	m.terminalSessions.Remove(sessionID)

	return nil
}

// ─── DetectShell ───────────────────────────────────────────

func (m *Manager) DetectShell(namespace, pod, container string) (string, error) {
	shells := []string{"/bin/bash", "/bin/sh", "/bin/ash"}

	for _, shell := range shells {
		_, err := m.execRaw(namespace, pod, container, []string{shell, "-c", "echo ok"})
		if err == nil {
			return shell, nil
		}
	}

	return "/bin/sh", nil
}

// ─── execRaw ───────────────────────────────────────────────

func (m *Manager) execRaw(namespace, pod, container string, command []string) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}

	req := conn.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(conn.Config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", stderr.String())
		}
		return "", err
	}

	return stdout.String(), nil
}
