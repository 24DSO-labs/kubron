package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalSession struct {
	Namespace string
	Pod       string
	Container string
	Shell     string
	Cwd       string
}

type TerminalOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

// DetectShell определяет какой shell доступен в контейнере
func (m *Manager) DetectShell(namespace, pod, container string) (string, error) {
	shells := []string{"/bin/bash", "/bin/sh", "/bin/ash"}

	for _, shell := range shells {
		_, err := m.execRaw(namespace, pod, container, []string{shell, "-c", "echo ok"})
		if err == nil {
			return shell, nil
		}
	}

	return "/bin/sh", nil // fallback
}

// ExecTerminal выполняет команду с полным выводом
func (m *Manager) ExecTerminal(namespace, pod, container, shell, command string) (*TerminalOutput, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	// Оборачиваем команду чтобы получить exit code
	wrappedCmd := fmt.Sprintf(`%s 2>&1; echo ":::EXIT_CODE:::$?"`, command)

	req := conn.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{shell, "-c", wrappedCmd},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(conn.Config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("ошибка создания executor: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	output := stdout.String()
	exitCode := 0

	// Извлекаем exit code
	if idx := strings.LastIndex(output, ":::EXIT_CODE:::"); idx >= 0 {
		codeStr := strings.TrimSpace(output[idx+15:])
		output = output[:idx]
		fmt.Sscanf(codeStr, "%d", &exitCode)
	}

	if err != nil && exitCode == 0 {
		exitCode = 1
	}

	return &TerminalOutput{
		Stdout:   output,
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// GetWorkingDirectory получает текущую директорию
func (m *Manager) GetWorkingDirectory(namespace, pod, container, shell string) string {
	result, err := m.execRaw(namespace, pod, container, []string{shell, "-c", "pwd"})
	if err != nil {
		return "/"
	}
	return strings.TrimSpace(result)
}

// GetEnvInfo получает информацию об окружении
func (m *Manager) GetEnvInfo(namespace, pod, container, shell string) (string, error) {
	result, err := m.execRaw(namespace, pod, container, []string{shell, "-c",
		`echo "USER=$(whoami 2>/dev/null || echo unknown)" && echo "HOME=$HOME" && echo "SHELL=$SHELL" && echo "HOSTNAME=$(hostname 2>/dev/null || echo unknown)" && echo "PWD=$PWD"`,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// TabComplete выполняет автодополнение
func (m *Manager) TabComplete(namespace, pod, container, shell, partial string) ([]string, error) {
	cmd := fmt.Sprintf(`compgen -f "%s" 2>/dev/null || ls -1 "%s"* 2>/dev/null`, partial, partial)
	result, err := m.execRaw(namespace, pod, container, []string{shell, "-c", cmd})
	if err != nil {
		return nil, err
	}

	var completions []string
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if line != "" {
			completions = append(completions, line)
		}
	}
	return completions, nil
}

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
