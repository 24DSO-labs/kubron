// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Хеш хранится строкой вида: pbkdf2-sha256$<итерации>$<соль base64>$<хеш base64>.
// Число итераций лежит внутри строки, поэтому его можно увеличивать, не ломая старые пароли:
// при успешном входе устаревший хеш пересчитывается (см. needsRehash).
const (
	hashScheme   = "pbkdf2-sha256"
	saltLength   = 16
	keyLength    = 32
	maxIter      = 10_000_000 // защита от подменённого файла с абсурдным числом итераций
	minPassRunes = 8
	maxPassBytes = 256
)

// pbkdf2Iterations — актуальная стоимость хеширования (рекомендация OWASP для PBKDF2-HMAC-SHA256).
// Переменная, а не константа, только чтобы тесты могли ускорять хеширование.
var pbkdf2Iterations = 600_000

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

// ValidateUsername проверяет формат логина.
func ValidateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return errors.New("логин: 3–32 символа, допустимы латиница, цифры, точка, дефис и подчёркивание")
	}
	return nil
}

// ValidatePassword проверяет требования к паролю.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPassRunes {
		return fmt.Errorf("пароль должен содержать не менее %d символов", minPassRunes)
	}
	if len(password) > maxPassBytes {
		return fmt.Errorf("пароль слишком длинный (максимум %d байт)", maxPassBytes)
	}
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("не удалось получить случайные данные: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, keyLength)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		hashScheme,
		strconv.Itoa(pbkdf2Iterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$"), nil
}

// verifyPassword сравнивает пароль с сохранённым хешем за постоянное время.
// needsRehash == true, если хеш посчитан с устаревшими параметрами.
func verifyPassword(password, encoded string) (ok, needsRehash bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != hashScheme {
		return false, false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 || iter > maxIter {
		return false, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return false, false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false, false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false, false
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false
	}
	return true, iter < pbkdf2Iterations || len(want) != keyLength
}
