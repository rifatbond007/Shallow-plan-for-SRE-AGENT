package codebase

import (
	"os"
	"path/filepath"
	"strings"
)

type Scanner struct {
	Root string
}

func NewScanner(root string) *Scanner {
	return &Scanner{Root: root}
}

func (s *Scanner) GoFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
