package k8s

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Rollback и uninstall ждут готовности ресурсов; больше — считаем операцию зависшей.
const helmActionTimeout = 5 * time.Minute

// helmConfig создаёт action.Configuration для активного кластера и namespace релиза.
// Подключаемся тем же kubeconfig и контекстом, что и основное подключение приложения.
func (m *Manager) helmConfig(namespace string) (*action.Configuration, error) {
	id := m.GetActiveContext()
	if id == "" {
		return nil, errors.New("нет активного подключения")
	}
	meta, err := LoadClusterMeta(id)
	if err != nil {
		return nil, fmt.Errorf("метаданные кластера не найдены: %w", err)
	}
	kubeconfig := GetClusterKubeconfigPath(id)
	if kubeconfig == "" {
		return nil, fmt.Errorf("kubeconfig не найден для кластера %s", id)
	}

	flags := genericclioptions.NewConfigFlags(false)
	flags.KubeConfig = &kubeconfig
	flags.Namespace = &namespace
	if meta.ContextName != "" {
		ctxName := meta.ContextName
		flags.Context = &ctxName
	}

	cfg := new(action.Configuration)
	if err := cfg.Init(flags, namespace, "secret"); err != nil {
		return nil, fmt.Errorf("не удалось инициализировать Helm: %w", err)
	}
	return cfg, nil
}

// RollbackHelmRelease откатывает релиз к ревизии. Helm создаёт НОВУЮ ревизию с содержимым старой.
func (m *Manager) RollbackHelmRelease(namespace, name string, revision int) error {
	if err := validateHelmRef(namespace, name); err != nil {
		return err
	}
	if revision <= 0 {
		return fmt.Errorf("некорректная ревизия %d", revision)
	}
	cfg, err := m.helmConfig(namespace)
	if err != nil {
		return err
	}
	a := action.NewRollback(cfg)
	a.Version = revision
	a.Timeout = helmActionTimeout
	a.WaitStrategy = kube.StatusWatcherStrategy
	if err := a.Run(name); err != nil {
		return fmt.Errorf("откат %s/%s к ревизии %d: %w", namespace, name, revision, err)
	}
	return nil
}

// UninstallHelmRelease удаляет релиз и его ресурсы. keepHistory оставляет записи ревизий
// (тогда имя релиза можно переиспользовать только через upgrade --install).
func (m *Manager) UninstallHelmRelease(namespace, name string, keepHistory bool) error {
	if err := validateHelmRef(namespace, name); err != nil {
		return err
	}
	cfg, err := m.helmConfig(namespace)
	if err != nil {
		return err
	}
	a := action.NewUninstall(cfg)
	a.KeepHistory = keepHistory
	a.Timeout = helmActionTimeout
	a.WaitStrategy = kube.StatusWatcherStrategy
	if _, err := a.Run(name); err != nil {
		return fmt.Errorf("удаление %s/%s: %w", namespace, name, err)
	}
	return nil
}

// HelmRevisionDiff — различия двух ревизий. Манифест уже с замаскированными Secret.
type HelmRevisionDiff struct {
	From     int    `json:"from"`
	To       int    `json:"to"`
	Values   string `json:"values"`   // user-supplied values, unified diff; "" — без различий
	Manifest string `json:"manifest"` // unified diff манифестов; "" — без различий
}

// GetHelmRevisionDiff сравнивает ревизии from → to. Требует права на секреты:
// в values могут быть пароли, поэтому метод защищён так же, как GetHelmReleaseContent.
func (m *Manager) GetHelmRevisionDiff(namespace, name string, from, to int) (*HelmRevisionDiff, error) {
	a, err := m.GetHelmReleaseContent(namespace, name, from)
	if err != nil {
		return nil, err
	}
	b, err := m.GetHelmReleaseContent(namespace, name, to)
	if err != nil {
		return nil, err
	}
	return &HelmRevisionDiff{
		From: from, To: to,
		Values:   textDiff(a.Values, b.Values),
		Manifest: textDiff(a.Manifest, b.Manifest),
	}, nil
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// textDiff — unified diff двух текстов с защитой от слишком больших входов (LCS квадратичен).
func textDiff(a, b string) string {
	la, lb := splitLines(a), splitLines(b)
	if len(la) > argoDiffMaxLines || len(lb) > argoDiffMaxLines {
		return "(текст слишком большой для построчного сравнения)"
	}
	d := unifiedDiff(la, lb, argoDiffContext)
	if len(d) > argoDiffMaxTextBytes {
		d = d[:argoDiffMaxTextBytes] + "\n… (diff обрезан)"
	}
	return d
}
