// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package secretbox

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func keyString(id string) string {
	b := make([]byte, KeySize)
	_, _ = rand.Read(b)
	return id + ":" + base64.StdEncoding.EncodeToString(b)
}

func mustRing(t *testing.T, cur string, prev ...string) *Keyring {
	t.Helper()
	kr, err := ParseKeyring(cur, prev...)
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

func TestSealOpenRoundTrip(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	secret := []byte("apiVersion: v1\nkind: Config\n")
	ct, id, err := kr.Seal(secret, []byte("cluster/prod/kubeconfig"))
	if err != nil || id != "k1" {
		t.Fatalf("%v %q", err, id)
	}
	if bytes.Contains(ct, []byte("apiVersion")) {
		t.Fatal("в шифротексте открытый текст")
	}
	got, err := kr.Open(ct, id, []byte("cluster/prod/kubeconfig"))
	if err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestEmptyPlaintextRoundTrips(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	ct, id, _ := kr.Seal(nil, nil)
	if got, err := kr.Open(ct, id, nil); err != nil || len(got) != 0 {
		t.Fatalf("%q %v", got, err)
	}
}

func TestNonceIsFreshEachTime(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	a, _, _ := kr.Seal([]byte("same"), nil)
	b, _, _ := kr.Seal([]byte("same"), nil)
	if bytes.Equal(a, b) {
		t.Fatal("одинаковый шифротекст для одинаковых данных: nonce повторился")
	}
}

func TestAADBindsCiphertextToRecord(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	ct, id, _ := kr.Seal([]byte("secret"), []byte("cluster/a"))
	if _, err := kr.Open(ct, id, []byte("cluster/b")); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("шифротекст из другой записи должен отклоняться: %v", err)
	}
}

func TestTamperingAndTruncationAreRejected(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	ct, id, _ := kr.Seal([]byte("secret"), nil)
	bad := append([]byte(nil), ct...)
	bad[len(bad)-1] ^= 1
	if _, err := kr.Open(bad, id, nil); !errors.Is(err, ErrDecrypt) {
		t.Errorf("подмена: %v", err)
	}
	for _, n := range []int{0, 5, 12, 20} {
		if _, err := kr.Open(ct[:min(n, len(ct)-1)], id, nil); !errors.Is(err, ErrDecrypt) {
			t.Errorf("обрезка до %d: %v", n, err)
		}
	}
}

func TestKeyRotation(t *testing.T) {
	oldS, newS := keyString("2026-01"), keyString("2026-07")
	oldRing := mustRing(t, oldS)
	ct, id, _ := oldRing.Seal([]byte("secret"), nil)

	// Новый ключ — текущий, старый остался для чтения.
	ring := mustRing(t, newS, oldS)
	if got, err := ring.Open(ct, id, nil); err != nil || string(got) != "secret" {
		t.Fatalf("старая запись при ротации: %q %v", got, err)
	}
	if !ring.NeedsRotation(id) || ring.NeedsRotation(ring.CurrentID()) {
		t.Error("NeedsRotation")
	}
	ct2, id2, _ := ring.Seal([]byte("secret"), nil)
	if id2 != "2026-07" {
		t.Errorf("новые записи шифруются текущим ключом, получили %q", id2)
	}
	// Без старого ключа старая запись не читается — потеря ключа = потеря секретов.
	if _, err := mustRing(t, newS).Open(ct, id, nil); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("без старого ключа: %v", err)
	}
	if _, err := mustRing(t, newS).Open(ct2, id2, nil); err != nil {
		t.Errorf("запись новым ключом: %v", err)
	}
}

func TestWrongKeyWithSameIDIsRejected(t *testing.T) {
	ct, id, _ := mustRing(t, keyString("k1")).Seal([]byte("secret"), nil)
	if _, err := mustRing(t, keyString("k1")).Open(ct, id, nil); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("другой ключ под тем же именем: %v", err)
	}
}

func TestParseKeyValidation(t *testing.T) {
	good := keyString("k1")
	if _, err := ParseKey(good); err != nil {
		t.Fatal(err)
	}
	// URL-вариант и вариант без паддинга тоже принимаются.
	raw := make([]byte, KeySize)
	_, _ = rand.Read(raw)
	for _, enc := range []string{base64.URLEncoding.EncodeToString(raw), base64.RawStdEncoding.EncodeToString(raw)} {
		if _, err := ParseKey("k:" + enc); err != nil {
			t.Errorf("%q: %v", enc, err)
		}
	}
	bad := []string{
		"", "нет-двоеточия", ":" + good[3:], "k1:!!!", "k1:" + base64.StdEncoding.EncodeToString(make([]byte, 16)),
		"bad id:" + good[3:], strings.Repeat("a", 33) + ":" + good[3:],
	}
	for _, s := range bad {
		if _, err := ParseKey(s); err == nil {
			t.Errorf("%q должен отклоняться", s)
		}
	}
}

func TestParseKeyringRequiresKeyAndUniqueIDs(t *testing.T) {
	if _, err := ParseKeyring("  "); !errors.Is(err, ErrNoKey) {
		t.Errorf("пустой ключ: %v", err)
	}
	k := keyString("k1")
	if _, err := ParseKeyring(k, k); err == nil {
		t.Error("повтор идентификатора ключа должен отклоняться")
	}
	if _, err := ParseKeyring(k, "", "  "); err != nil {
		t.Errorf("пустые прежние ключи игнорируются: %v", err)
	}
}

func TestConcurrentUse(t *testing.T) {
	kr := mustRing(t, keyString("k1"))
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 200; j++ {
				ct, id, _ := kr.Seal([]byte("x"), nil)
				if _, err := kr.Open(ct, id, nil); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}
