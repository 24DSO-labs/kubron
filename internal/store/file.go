package store

import (
	"analog/internal/audit"
	"analog/internal/auth"
	"analog/internal/k8s"
)

// fileStore — хранилище десктопа: файлы в каталоге данных приложения (~/.kubron).
type fileStore struct {
	dir string
}

// NewFile возвращает файловое хранилище. Пользователи и журнал лежат в dir; токены Argo CD
// — в собственном каталоге k8s (argoTokenDir), как раньше.
func NewFile(dir string) Store { return fileStore{dir: dir} }

func (s fileStore) Users() auth.Backend        { return auth.NewFileBackend(s.dir) }
func (s fileStore) Audit() audit.Backend       { return audit.NewFileBackend(s.dir) }
func (s fileStore) ArgoTokens() k8s.TokenStore { return k8s.NewFileTokenStore() }
