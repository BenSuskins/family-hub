package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDomainAndApplicationDependencies(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find test directory")
	}
	internal := filepath.Dir(filepath.Dir(file))
	for _, layer := range []string{"domain", "application"} {
		err := filepath.WalkDir(filepath.Join(internal, layer), func(path string, entry fs.DirEntry, walkError error) error {
			if walkError != nil {
				return walkError
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imported := range parsed.Imports {
				name, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}
				prohibited := name == "database/sql" || name == "net/http" || strings.Contains(name, "/internal/repository") || strings.Contains(name, "/internal/handlers") || strings.Contains(name, "/internal/server") || strings.Contains(name, "/internal/services")
				if layer == "domain" && strings.Contains(name, "/internal/application/") {
					prohibited = true
				}
				if prohibited {
					t.Errorf("%s imports adapter or transport package %s", path, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
