package audit

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// fileBackend — JSON Lines в файле audit.log с ротацией:
// audit.log → audit.log.1 → … → audit.log.<keepFiles-1>, самый старый удаляется.
type fileBackend struct {
	dir     string
	maxSize int64
}

func newFileBackend(dir string) *fileBackend {
	return &fileBackend{dir: dir, maxSize: maxFileSize}
}

func (b *fileBackend) path(i int) string {
	if i == 0 {
		return filepath.Join(b.dir, fileName)
	}
	return filepath.Join(b.dir, fmt.Sprintf("%s.%d", fileName, i))
}

func (b *fileBackend) Append(line []byte) error {
	line = append(append([]byte(nil), line...), '\n')

	if err := os.MkdirAll(b.dir, 0o700); err != nil {
		return err
	}
	if st, err := os.Stat(b.path(0)); err == nil && st.Size()+int64(len(line)) > b.maxSize {
		if err := b.rotate(); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(b.path(0), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (b *fileBackend) rotate() error {
	if err := os.Remove(b.path(keepFiles - 1)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for i := keepFiles - 2; i >= 0; i-- {
		if err := os.Rename(b.path(i), b.path(i+1)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (b *fileBackend) ScanNewestFirst(fn func(line []byte) bool) error {
	for i := 0; i < keepFiles; i++ {
		lines, err := readLines(b.path(i))
		if err != nil {
			return err
		}
		for j := len(lines) - 1; j >= 0; j-- {
			if !fn(lines[j]) {
				return nil
			}
		}
	}
	return nil
}

// readLines читает непустые строки файла; отсутствие файла — не ошибка.
func readLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	// ENOTDIR: на пути лежит файл вместо каталога — журнала там тоже нет.
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать журнал: %w", err)
	}
	defer f.Close()

	var out [][]byte
	r := bufio.NewReaderSize(f, 64<<10)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			out = append(out, line)
		}
		if err != nil {
			break
		}
	}
	return out, nil
}
