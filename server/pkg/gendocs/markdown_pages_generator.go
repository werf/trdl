package gendocs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MarkdownPagesGenerator struct {
	Dir string
}

func NewMarkdownPagesGenerator(dir string) *MarkdownPagesGenerator {
	return &MarkdownPagesGenerator{
		Dir: dir,
	}
}

func (w *MarkdownPagesGenerator) HandlePath(pathPattern string, doc []byte) error {
	fsPath, err := PathPatternToFilesystemMarkdownPath(pathPattern)
	if err != nil {
		return err
	}

	f := filepath.Join(w.Dir, fsPath)
	if err := os.MkdirAll(filepath.Dir(f), os.ModePerm); err != nil {
		return fmt.Errorf("unable to make dir %q: %s", filepath.Dir(f), err)
	}
	if err := os.WriteFile(f, append(doc, '\n'), 0644); err != nil {
		return fmt.Errorf("unable to write file %q: %s", f, err)
	}

	return nil
}

func (w *MarkdownPagesGenerator) Close() error {
	return nil
}

func PathPatternToFilesystemMarkdownPath(pattern string) (string, error) {
	// fmt.Printf("INPUT PATTERN: %q\n", pattern)
	if pattern == "/" {
		return "index.md", nil
	}

	parts := strings.Split(pattern, "/")
	var newParts []string

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			newParts = append(newParts, strings.TrimPrefix(part, ":"))
		} else {
			newParts = append(newParts, part)
		}
	}

	return filepath.Join(newParts...) + ".md", nil
}
