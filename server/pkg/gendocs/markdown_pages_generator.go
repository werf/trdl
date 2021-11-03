package gendocs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp/syntax"
	"strings"

	regen "github.com/zach-klippenstein/goregen"
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
	if err := os.WriteFile(f, append(doc, '\n'), os.ModePerm); err != nil {
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

	cleanPattern := pattern
	cleanPattern = strings.TrimPrefix(cleanPattern, "^")
	cleanPattern = strings.TrimSuffix(cleanPattern, "$")
	cleanPattern = strings.TrimPrefix(cleanPattern, "/")
	cleanPattern = strings.TrimSuffix(cleanPattern, "/")
	// fmt.Printf("CLEAN PATTERN: %q\n", cleanPattern)

	generator, err := regen.NewGenerator(cleanPattern, generatorArgs())
	if err != nil {
		return "", fmt.Errorf("bad pattern given: %s", err)
	}

	path := generator.Generate()
	// fmt.Printf("PATH: %q\n", path)
	pathParts := strings.Split(path, "/")
	// fmt.Printf("PATH PARTS: %#v\n", pathParts)

	patternParts := strings.Split(cleanPattern, "/")
	// fmt.Printf("PATTERN PARTS: %#v\n", patternParts)

	regexp, err := syntax.Parse(cleanPattern, syntax.Perl)
	if err != nil {
		return "", fmt.Errorf("bad pattern %q: %s", pattern, err)
	}

	getPatternName := func(n int) string {
		if n+1 < len(regexp.Sub) {
			if regexp.Sub[n+1].Op == syntax.OpCapture {
				return fmt.Sprintf("%s_pattern", regexp.Sub[n+1].Name)
			}
		}
		return fmt.Sprintf("pattern_%d", n)
	}

	var resPathParts []string
	patternNum := 0
	for i := 0; i < len(pathParts); i++ {
		if pathParts[i] == "" {
			continue
		}

		if i >= len(patternParts) || pathParts[i] != patternParts[i] {
			resPathParts = append(resPathParts, getPatternName(patternNum))
			patternNum++
		} else {
			resPathParts = append(resPathParts, pathParts[i])
		}
	}

	resPath := fmt.Sprintf("%s.md", filepath.Join(resPathParts...))
	return resPath, nil
}
