// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	argoSecretTypeLabel = "argocd.argoproj.io/secret-type"
	argoSecretRepo      = "repository"
	argoSecretCluster   = "cluster"
)

// ArgoRepositoryInfo — санитизированное описание репозитория Argo CD (Secret с лейблом secret-type=repository).
//
// БЕЗОПАСНОСТЬ: в структуре только явно перечисленные безопасные поля. Пароли, токены, SSH-ключи,
// TLS-ключи, приватные ключи GitHub App, а также аннотации/лейблы Secret (в аннотации
// kubectl.kubernetes.io/last-applied-configuration может лежать копия данных) наружу не отдаются.
// Не добавляйте сюда поля «как есть» из Secret: новое поле должно проходить через тест TestArgoSettingsNoSecretLeak.
type ArgoRepositoryInfo struct {
	Name      string `json:"name"`      // имя Secret
	Namespace string `json:"namespace"` // namespace Secret (= namespace Argo CD)
	Age       string `json:"age"`

	RepoName    string `json:"repoName"` // data.name (отображаемое имя)
	URL         string `json:"url"`      // без user:password@
	Type        string `json:"type"`     // git | helm | oci
	Project     string `json:"project"`
	AuthType    string `json:"authType"` // anonymous | password | token | ssh | tls | github-app
	Insecure    bool   `json:"insecure"`
	EnableLFS   bool   `json:"enableLfs"`
	EnableOCI   bool   `json:"enableOCI"`
	ForceBasic  bool   `json:"forceHttpBasicAuth"`
	ProxyConfig bool   `json:"proxy"` // задан proxy (сам адрес не отдаётся: может содержать учётные данные)
}

// ArgoClusterInfo — санитизированное описание кластера Argo CD (Secret с лейблом secret-type=cluster).
// Поле config (токены, сертификаты, exec-провайдеры) наружу не отдаётся: из него выводится только тип аутентификации.
type ArgoClusterInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`

	ClusterName      string `json:"clusterName"` // data.name
	Server           string `json:"server"`      // без user:password@
	Project          string `json:"project"`
	AuthType         string `json:"authType"` // none | token | tls | basic | aws | exec | unknown
	Insecure         bool   `json:"insecure"` // tlsClientConfig.insecure
	Namespaces       string `json:"namespaces"`
	ClusterResources bool   `json:"clusterResources"`
	InCluster        bool   `json:"inCluster"`
}

// GetArgoRepositories возвращает репозитории Argo CD из namespace (пустой — из всех).
func (m *Manager) GetArgoRepositories(namespace string) ([]ArgoRepositoryInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return listArgoRepositories(ctx, conn.Client, namespace)
}

// GetArgoClusters возвращает кластеры Argo CD из namespace (пустой — из всех).
func (m *Manager) GetArgoClusters(namespace string) ([]ArgoClusterInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return listArgoClusters(ctx, conn.Client, namespace)
}

func listArgoSecrets(ctx context.Context, c kubernetes.Interface, namespace, secretType string) ([]corev1.Secret, error) {
	list, err := c.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: argoSecretTypeLabel + "=" + secretType,
	})
	if err != nil {
		return nil, fmt.Errorf("list argo %s secrets: %w", secretType, err)
	}
	sort.SliceStable(list.Items, func(i, j int) bool {
		a, b := list.Items[i], list.Items[j]
		if a.Namespace != b.Namespace {
			return a.Namespace < b.Namespace
		}
		return a.Name < b.Name
	})
	return list.Items, nil
}

func listArgoRepositories(ctx context.Context, c kubernetes.Interface, namespace string) ([]ArgoRepositoryInfo, error) {
	secrets, err := listArgoSecrets(ctx, c, namespace, argoSecretRepo)
	if err != nil {
		return nil, err
	}
	out := make([]ArgoRepositoryInfo, 0, len(secrets))
	for i := range secrets {
		out = append(out, parseArgoRepository(&secrets[i]))
	}
	return out, nil
}

func listArgoClusters(ctx context.Context, c kubernetes.Interface, namespace string) ([]ArgoClusterInfo, error) {
	secrets, err := listArgoSecrets(ctx, c, namespace, argoSecretCluster)
	if err != nil {
		return nil, err
	}
	out := make([]ArgoClusterInfo, 0, len(secrets))
	for i := range secrets {
		out = append(out, parseArgoCluster(&secrets[i]))
	}
	return out, nil
}

func secretStr(s *corev1.Secret, key string) string { return strings.TrimSpace(string(s.Data[key])) }
func secretHas(s *corev1.Secret, key string) bool   { return len(s.Data[key]) > 0 }
func secretBool(s *corev1.Secret, key string) bool {
	return strings.EqualFold(secretStr(s, key), "true")
}

func parseArgoRepository(s *corev1.Secret) ArgoRepositoryInfo {
	info := ArgoRepositoryInfo{
		Name:        s.Name,
		Namespace:   s.Namespace,
		RepoName:    secretStr(s, "name"),
		URL:         stripURLUserinfo(secretStr(s, "url")),
		Type:        orDefault(secretStr(s, "type"), "git"),
		Project:     secretStr(s, "project"),
		Insecure:    secretBool(s, "insecure"),
		EnableLFS:   secretBool(s, "enableLfs"),
		EnableOCI:   secretBool(s, "enableOCI"),
		ForceBasic:  secretBool(s, "forceHttpBasicAuth"),
		ProxyConfig: secretHas(s, "proxy"),
	}
	if !s.CreationTimestamp.IsZero() {
		info.Age = formatAge(s.CreationTimestamp.Time)
	}
	switch {
	case secretHas(s, "sshPrivateKey"):
		info.AuthType = "ssh"
	case secretHas(s, "githubAppPrivateKey"):
		info.AuthType = "github-app"
	case secretHas(s, "tlsClientCertData") || secretHas(s, "tlsClientCertKey"):
		info.AuthType = "tls"
	case secretHas(s, "bearerToken"):
		info.AuthType = "token"
	case secretHas(s, "password") || secretHas(s, "username"):
		info.AuthType = "password"
	default:
		info.AuthType = "anonymous"
	}
	return info
}

// argoClusterConfig — только те поля config кластера, по которым определяется тип аутентификации.
// Значения не сохраняются в ответ: json.Unmarshal нужен лишь чтобы узнать, какие ключи непустые.
type argoClusterConfig struct {
	Username        string           `json:"username"`
	Password        string           `json:"password"`
	BearerToken     string           `json:"bearerToken"`
	TLSClientConfig *argoClusterTLS  `json:"tlsClientConfig"`
	AWSAuthConfig   *json.RawMessage `json:"awsAuthConfig"`
	ExecProvider    *json.RawMessage `json:"execProviderConfig"`
}

type argoClusterTLS struct {
	Insecure bool   `json:"insecure"`
	CertData string `json:"certData"`
	KeyData  string `json:"keyData"`
}

func parseArgoCluster(s *corev1.Secret) ArgoClusterInfo {
	server := stripURLUserinfo(secretStr(s, "server"))
	info := ArgoClusterInfo{
		Name:             s.Name,
		Namespace:        s.Namespace,
		ClusterName:      secretStr(s, "name"),
		Server:           server,
		Project:          secretStr(s, "project"),
		Namespaces:       secretStr(s, "namespaces"),
		ClusterResources: secretBool(s, "clusterResources"),
		InCluster:        server == "https://kubernetes.default.svc",
		AuthType:         "none",
	}
	if !s.CreationTimestamp.IsZero() {
		info.Age = formatAge(s.CreationTimestamp.Time)
	}
	if !secretHas(s, "config") {
		return info
	}
	var cfg argoClusterConfig
	if err := json.Unmarshal(s.Data["config"], &cfg); err != nil {
		info.AuthType = "unknown"
		return info
	}
	if cfg.TLSClientConfig != nil {
		info.Insecure = cfg.TLSClientConfig.Insecure
	}
	switch {
	case cfg.AWSAuthConfig != nil:
		info.AuthType = "aws"
	case cfg.ExecProvider != nil:
		info.AuthType = "exec"
	case cfg.BearerToken != "":
		info.AuthType = "token"
	case cfg.TLSClientConfig != nil && (cfg.TLSClientConfig.CertData != "" || cfg.TLSClientConfig.KeyData != ""):
		info.AuthType = "tls"
	case cfg.Username != "" || cfg.Password != "":
		info.AuthType = "basic"
	}
	return info
}
