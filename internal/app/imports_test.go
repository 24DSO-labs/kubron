package app

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ядро не должно зависеть от Wails: платформенные возможности приходят через Platform.
func TestCoreDoesNotImportWails(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("нет файлов: %v", err)
	}
	fset := token.NewFileSet()
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("разбор %s: %v", f, err)
		}
		for _, imp := range file.Imports {
			if strings.Contains(imp.Path.Value, "github.com/wailsapp/wails") {
				t.Errorf("%s импортирует %s: используйте интерфейс Platform", f, imp.Path.Value)
			}
		}
	}
}
