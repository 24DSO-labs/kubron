// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func tlsSecret(t *testing.T, name string, notAfter time.Time, managed bool) corev1.Secret {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name + ".example.com"},
		NotBefore:    notAfter.Add(-90 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{corev1.TLSCertKey: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})},
	}
	if managed {
		s.Annotations = map[string]string{certManagerAnnotation: name}
	}
	return s
}

func TestScanTLSSecrets(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	garbage := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "garbage", Namespace: "ns"}, Data: map[string][]byte{corev1.TLSCertKey: []byte("not pem")}}

	scan := scanTLSSecrets([]corev1.Secret{
		tlsSecret(t, "soon", now.Add(3*day+time.Hour), false),
		tlsSecret(t, "month", now.Add(20*day+time.Hour), false),
		tlsSecret(t, "managed-renewing", now.Add(20*day+time.Hour), true), // cert-manager ещё обновит: не шумим
		tlsSecret(t, "managed-stuck", now.Add(5*day+time.Hour), true),
		tlsSecret(t, "far", now.Add(60*day), false),
		tlsSecret(t, "expired", now.Add(-2*day), false),
		garbage,
	}, now)

	if scan.Checked != 6 {
		t.Errorf("checked=%d, want 6 (мусор не считается)", scan.Checked)
	}
	var names []string
	for _, c := range scan.Expiring {
		names = append(names, c.Name)
	}
	want := []string{"expired", "soon", "managed-stuck", "month"}
	if len(names) != len(want) {
		t.Fatalf("expiring=%v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("expiring[%d]=%s, want %s", i, names[i], want[i])
		}
	}
	if scan.Expiring[0].DaysLeft >= 0 || scan.Expiring[1].DaysLeft != 3 {
		t.Errorf("daysLeft: %+v", scan.Expiring)
	}
}
