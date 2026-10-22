package k8s

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ErrArgoTokenNotSet — для активного кластера не задан токен Argo CD API.
var ErrArgoTokenNotSet = errors.New("токен Argo CD API не задан")

const argoTokenMaxLen = 8192

// argoTokenDir — каталог токенов; переменная нужна тестам (по умолчанию ~/.kubron/argocd).
var argoTokenDir = func() string { return filepath.Join(GetKubronDir(), "argocd") }

// ArgoAPIStatus — состояние опционального провайдера Argo CD API. Сам токен фронтенду не отдаётся никогда.
type ArgoAPIStatus struct {
	Configured bool `json:"configured"`
}

// БЕЗОПАСНОСТЬ. Токен хранится так же, как kubeconfig-файлы кластеров: файл 0600 в каталоге 0700 внутри
// ~/.kubron (отдельного хранилища секретов в приложении нет). Имя файла — хеш идентификатора кластера,
// поэтому оно не зависит от символов в имени и не позволяет выйти из каталога. Токен нельзя:
// возвращать во фронтенд, писать в логи, передавать в a.authz (попадёт в аудит) и включать в тексты ошибок.
func argoTokenPath(clusterID string) (string, error) {
	if clusterID == "" {
		return "", errors.New("нет активного кластера")
	}
	sum := sha256.Sum256([]byte(clusterID))
	return filepath.Join(argoTokenDir(), hex.EncodeToString(sum[:])+".token"), nil
}

// normalizeArgoToken убирает пробелы по краям и отклоняет заведомо некорректные значения.
// Сообщения об ошибках намеренно не содержат само значение.
func normalizeArgoToken(token string) (string, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return "", errors.New("токен пустой")
	}
	if len(t) > argoTokenMaxLen {
		return "", errors.New("токен слишком длинный")
	}
	for _, r := range t {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", errors.New("токен содержит пробелы или управляющие символы")
		}
	}
	return t, nil
}

func saveArgoToken(clusterID, token string) error {
	t, err := normalizeArgoToken(token)
	if err != nil {
		return err
	}
	path, err := argoTokenPath(clusterID)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("не удалось создать каталог токенов: %w", err)
	}
	// Атомарная запись: во временный файл (CreateTemp создаёт его с правами 0600) и переименование.
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("не удалось сохранить токен: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(t); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("не удалось сохранить токен: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("не удалось сохранить токен: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("не удалось сохранить токен: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("не удалось сохранить токен: %w", err)
	}
	return nil
}

func loadArgoToken(clusterID string) (string, error) {
	path, err := argoTokenPath(clusterID)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrArgoTokenNotSet
		}
		return "", fmt.Errorf("не удалось прочитать токен: %w", err)
	}
	t := strings.TrimSpace(string(data))
	if t == "" {
		return "", ErrArgoTokenNotSet
	}
	return t, nil
}

func deleteArgoToken(clusterID string) error {
	path, err := argoTokenPath(clusterID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("не удалось удалить токен: %w", err)
	}
	return nil
}

// GetArgoAPIStatus сообщает, задан ли токен для активного кластера.
func (m *Manager) GetArgoAPIStatus() (*ArgoAPIStatus, error) {
	id := m.GetActiveContext()
	if id == "" {
		return nil, errors.New("нет активного подключения")
	}
	_, err := loadArgoToken(id)
	switch {
	case err == nil:
		return &ArgoAPIStatus{Configured: true}, nil
	case errors.Is(err, ErrArgoTokenNotSet):
		return &ArgoAPIStatus{Configured: false}, nil
	default:
		return nil, err
	}
}

// SetArgoAPIToken сохраняет токен Argo CD API для активного кластера.
func (m *Manager) SetArgoAPIToken(token string) error {
	id := m.GetActiveContext()
	if id == "" {
		return errors.New("нет активного подключения")
	}
	return saveArgoToken(id, token)
}

// DeleteArgoAPIToken удаляет токен активного кластера.
func (m *Manager) DeleteArgoAPIToken() error {
	id := m.GetActiveContext()
	if id == "" {
		return errors.New("нет активного подключения")
	}
	return deleteArgoToken(id)
}
