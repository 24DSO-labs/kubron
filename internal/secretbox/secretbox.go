// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package secretbox — шифрование секретов в хранилище сервиса (ADR-0008, задача 3.1c): kubeconfig'и,
// пароли oidc-password, токены Argo CD. AES-256-GCM, у каждого ключа идентификатор — он хранится рядом
// с шифротекстом, чтобы ключ можно было заменить (ротация): новые записи шифруются текущим ключом,
// старые читаются прежним, пока их не перешифруют.
//
// Пакет зависит только от стандартной библиотеки.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// KeySize — длина ключа AES-256 в байтах.
const KeySize = 32

var (
	// ErrNoKey — не задан ключ шифрования.
	ErrNoKey = errors.New("encryption key is not set")
	// ErrUnknownKey — шифротекст зашифрован ключом, которого нет в связке (потерян или не передан).
	ErrUnknownKey = errors.New("ciphertext was encrypted with an unknown key")
	// ErrDecrypt — шифротекст не удалось расшифровать: повреждён, подменён или относится к другой записи.
	ErrDecrypt = errors.New("failed to decrypt the secret")
)

var keyIDRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,32}$`)

// Key — именованный ключ.
type Key struct {
	ID    string
	Bytes [KeySize]byte
}

// ParseKey разбирает ключ в формате «идентификатор:base64» (32 байта; допустим стандартный и URL-вариант base64).
// Идентификатор попадает в хранилище открытым текстом и не должен быть секретом.
func ParseKey(s string) (Key, error) {
	id, enc, ok := strings.Cut(strings.TrimSpace(s), ":")
	if !ok || !keyIDRe.MatchString(id) {
		return Key{}, errors.New("encryption key must look like id:base64 (id: letters, digits, _ . -, up to 32 characters)")
	}
	raw, err := decodeBase64(enc)
	if err != nil {
		return Key{}, fmt.Errorf("encryption key %q is not valid base64: %w", id, err)
	}
	if len(raw) != KeySize {
		return Key{}, fmt.Errorf("encryption key %q must be %d bytes, got %d", id, KeySize, len(raw))
	}
	k := Key{ID: id}
	copy(k.Bytes[:], raw)
	return k, nil
}

func decodeBase64(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("unsupported encoding")
}

// Keyring — связка ключей: текущий (им шифруют) и прежние (ими только читают при ротации).
type Keyring struct {
	current Key
	byID    map[string]cipher.AEAD
	ids     map[string]struct{}
	curAEAD cipher.AEAD
}

// NewKeyring собирает связку: первым идёт текущий ключ, остальные — прежние. Идентификаторы уникальны.
func NewKeyring(current Key, previous ...Key) (*Keyring, error) {
	kr := &Keyring{current: current, byID: map[string]cipher.AEAD{}, ids: map[string]struct{}{}}
	for i, k := range append([]Key{current}, previous...) {
		if !keyIDRe.MatchString(k.ID) {
			return nil, errors.New("invalid encryption key id")
		}
		if _, dup := kr.byID[k.ID]; dup {
			return nil, fmt.Errorf("duplicate encryption key id %q", k.ID)
		}
		aead, err := newGCM(k.Bytes[:])
		if err != nil {
			return nil, err
		}
		kr.byID[k.ID] = aead
		kr.ids[k.ID] = struct{}{}
		if i == 0 {
			kr.curAEAD = aead
		}
	}
	return kr, nil
}

// ParseKeyring разбирает текущий ключ и список прежних (формат ParseKey). Пустой текущий ключ — ErrNoKey.
func ParseKeyring(current string, previous ...string) (*Keyring, error) {
	if strings.TrimSpace(current) == "" {
		return nil, ErrNoKey
	}
	cur, err := ParseKey(current)
	if err != nil {
		return nil, err
	}
	var prev []Key
	for _, p := range previous {
		if strings.TrimSpace(p) == "" {
			continue
		}
		k, err := ParseKey(p)
		if err != nil {
			return nil, err
		}
		prev = append(prev, k)
	}
	return NewKeyring(cur, prev...)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// CurrentID — идентификатор текущего ключа (им шифруются новые записи).
func (k *Keyring) CurrentID() string { return k.current.ID }

// Seal шифрует plaintext текущим ключом. aad привязывает шифротекст к записи (например,
// "cluster/<id>/kubeconfig"): подмена шифротекста между строками БД обнаруживается при расшифровке.
// Результат — nonce ‖ шифротекст; keyID сохраняется рядом.
func (k *Keyring) Seal(plaintext, aad []byte) (ciphertext []byte, keyID string, err error) {
	nonce := make([]byte, k.curAEAD.NonceSize(), k.curAEAD.NonceSize()+len(plaintext)+k.curAEAD.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", fmt.Errorf("random source unavailable: %w", err)
	}
	return k.curAEAD.Seal(nonce, nonce, plaintext, aad), k.current.ID, nil
}

// Open расшифровывает данные ключом keyID с теми же aad, что были при Seal.
func (k *Keyring) Open(ciphertext []byte, keyID string, aad []byte) ([]byte, error) {
	aead, ok := k.byID[keyID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, keyID)
	}
	n := aead.NonceSize()
	if len(ciphertext) < n+aead.Overhead() {
		return nil, ErrDecrypt
	}
	plain, err := aead.Open(nil, ciphertext[:n], ciphertext[n:], aad)
	if err != nil {
		return nil, ErrDecrypt // подробности (какой именно тег не сошёлся) наружу не отдаём
	}
	return plain, nil
}

// NeedsRotation сообщает, что запись зашифрована не текущим ключом и её стоит перешифровать.
func (k *Keyring) NeedsRotation(keyID string) bool { return keyID != k.current.ID }
