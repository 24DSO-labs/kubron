// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Маркеры, которых не должно быть ни в одном ответе.
var argoSecretMarkers = []string{
	"S3CRET-PASSWORD", "S3CRET-TOKEN", "S3CRET-SSH-KEY", "S3CRET-TLS-KEY", "S3CRET-GH-KEY",
	"S3CRET-ANNOTATION", "S3CRET-LABEL", "S3CRET-PROXY", "S3CRET-URLPASS", "S3CRET-CERT",
	"S3CRET-AWS", "S3CRET-EXEC",
}

func argoSecret(ns, name, typ string, data map[string]string) *corev1.Secret {
	d := map[string][]byte{}
	for k, v := range data {
		d[k] = []byte(v)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: ns,
			Labels: map[string]string{
				argoSecretTypeLabel: typ,
				"custom":            "S3CRET-LABEL",
			},
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"data":{"password":"S3CRET-ANNOTATION"}}`,
			},
		},
		Data: d,
	}
}

func TestArgoSettingsNoSecretLeak(t *testing.T) {
	c := fake.NewSimpleClientset(
		argoSecret("argocd", "repo-pass", argoSecretRepo, map[string]string{
			"url": "https://user:S3CRET-URLPASS@git.example.com/o/r.git", "username": "bob", "password": "S3CRET-PASSWORD",
			"bearerToken": "S3CRET-TOKEN", "sshPrivateKey": "S3CRET-SSH-KEY", "tlsClientCertKey": "S3CRET-TLS-KEY",
			"tlsClientCertData": "S3CRET-CERT", "githubAppPrivateKey": "S3CRET-GH-KEY", "proxy": "http://p:S3CRET-PROXY@proxy:3128",
		}),
		argoSecret("argocd", "cluster-prod", argoSecretCluster, map[string]string{
			"name": "prod", "server": "https://u:S3CRET-URLPASS@k8s.example.com",
			"config": `{"bearerToken":"S3CRET-TOKEN","password":"S3CRET-PASSWORD","tlsClientConfig":{"keyData":"S3CRET-TLS-KEY","certData":"S3CRET-CERT"},` +
				`"awsAuthConfig":{"clusterName":"S3CRET-AWS"},"execProviderConfig":{"command":"S3CRET-EXEC"}}`,
		}),
	)
	ctx := context.Background()
	repos, err := listArgoRepositories(ctx, c, "")
	if err != nil {
		t.Fatal(err)
	}
	clusters, err := listArgoClusters(ctx, c, "")
	if err != nil {
		t.Fatal(err)
	}
	raw1, _ := json.Marshal(repos)
	raw2, _ := json.Marshal(clusters)
	all := string(raw1) + string(raw2)
	for _, m := range argoSecretMarkers {
		if strings.Contains(all, m) {
			t.Errorf("секрет %q попал в ответ: %s", m, all)
		}
	}
	if len(repos) != 1 || len(clusters) != 1 {
		t.Fatalf("repos=%d clusters=%d", len(repos), len(clusters))
	}
	if repos[0].URL != "https://git.example.com/o/r.git" || clusters[0].Server != "https://k8s.example.com" {
		t.Errorf("userinfo не вырезан: %+v %+v", repos[0], clusters[0])
	}
}

func TestParseArgoRepositoryAuthType(t *testing.T) {
	cases := []struct {
		name string
		data map[string]string
		want string
	}{
		{"anonymous", map[string]string{"url": "https://x"}, "anonymous"},
		{"password", map[string]string{"username": "u", "password": "p"}, "password"},
		{"token", map[string]string{"bearerToken": "t"}, "token"},
		{"ssh", map[string]string{"sshPrivateKey": "k", "password": "p"}, "ssh"},
		{"tls", map[string]string{"tlsClientCertData": "c"}, "tls"},
		{"github-app", map[string]string{"githubAppPrivateKey": "k"}, "github-app"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseArgoRepository(argoSecret("argocd", "r", argoSecretRepo, tc.data))
			if got.AuthType != tc.want || got.Type != "git" {
				t.Errorf("got %+v, want auth %s", got, tc.want)
			}
		})
	}
}

func TestParseArgoClusterAuthType(t *testing.T) {
	cases := []struct {
		name   string
		config string
		want   string
	}{
		{"нет config", "", "none"},
		{"битый json", "{not json", "unknown"},
		{"token", `{"bearerToken":"x"}`, "token"},
		{"tls", `{"tlsClientConfig":{"certData":"a"}}`, "tls"},
		{"basic", `{"username":"u"}`, "basic"},
		{"aws", `{"awsAuthConfig":{}}`, "aws"},
		{"exec", `{"execProviderConfig":{}}`, "exec"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{"server": "https://kubernetes.default.svc", "clusterResources": "true"}
			if tc.config != "" {
				data["config"] = tc.config
			}
			got := parseArgoCluster(argoSecret("argocd", "c", argoSecretCluster, data))
			if got.AuthType != tc.want || !got.InCluster || !got.ClusterResources {
				t.Errorf("got %+v, want auth %s", got, tc.want)
			}
		})
	}
}

func TestListArgoSecretsFiltersByLabelAndNamespace(t *testing.T) {
	other := argoSecret("argocd", "plain", "something-else", map[string]string{"url": "x"})
	c := fake.NewSimpleClientset(
		argoSecret("argocd", "b", argoSecretRepo, map[string]string{"url": "https://b"}),
		argoSecret("ns2", "a", argoSecretRepo, map[string]string{"url": "https://a"}),
		other,
	)
	all, err := listArgoRepositories(context.Background(), c, "")
	if err != nil || len(all) != 2 || all[0].Namespace != "argocd" {
		t.Fatalf("all: %v %+v", err, all)
	}
	one, err := listArgoRepositories(context.Background(), c, "ns2")
	if err != nil || len(one) != 1 || one[0].Name != "a" {
		t.Fatalf("ns2: %v %+v", err, one)
	}
}
