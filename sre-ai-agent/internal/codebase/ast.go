package codebase

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func BuildIndex(root string) (*Index, error) {
	s := NewScanner(root)
	files, err := s.GoFiles()
	if err != nil {
		return nil, err
	}

	idx := &Index{
		Functions: make(map[string]Function),
		ByFile:    make(map[string][]string),
	}

	fset := token.NewFileSet()

	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		if isGenerated(data) {
			continue
		}

		f, err := parser.ParseFile(fset, filePath, data, parser.ParseComments)
		if err != nil {
			continue
		}

		pkgPath := f.Name.Name
		relPath := filepath.ToSlash(filePath)

		v := &visitor{
			fset:    fset,
			f:       f,
			pkgPath: pkgPath,
			file:    relPath,
			data:    data,
			idx:     idx,
		}
		v.walk()
	}

	return idx, nil
}

func isGenerated(data []byte) bool {
	return strings.Contains(string(data), "Code generated") && strings.Contains(string(data), "DO NOT EDIT")
}
