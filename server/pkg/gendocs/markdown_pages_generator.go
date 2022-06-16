package gendocs

import (
	"fmt"
	"os"
	"path/filepath"
)

type MarkdownPagesGenerator struct {
	Dir string
}

func NewMarkdownPagesGenerator(dir string) *MarkdownPagesGenerator {
	return &MarkdownPagesGenerator{
		Dir: dir,
	}
}

func (w *MarkdownPagesGenerator) HasFormatPathLink() bool {
	return false
}

func (w *MarkdownPagesGenerator) FormatPathLink(_ string) string {
	return ""
}

func (w *MarkdownPagesGenerator) HandlePath(pathPattern string, doc []byte) error {
	fsPath, err := FormatPathPatternAsFilesystemMarkdownPath(pathPattern)
	if err != nil {
		return err
	}

	f := filepath.Join(w.Dir, fsPath)
	if err := os.MkdirAll(filepath.Dir(f), os.ModePerm); err != nil {
		return fmt.Errorf("unable to make dir %q: %w", filepath.Dir(f), err)
	}
	if err := os.WriteFile(f, append(doc, '\n'), 0o644); err != nil {
		return fmt.Errorf("unable to write file %q: %w", f, err)
	}

	return nil
}

func (w *MarkdownPagesGenerator) Close() error {
	return nil
}
