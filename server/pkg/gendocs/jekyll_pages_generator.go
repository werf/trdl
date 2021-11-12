package gendocs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type JekyllPagesGenerator struct {
	markdownPagesGenerator *MarkdownPagesGenerator
	jekyllSidebar          *JekyllSidebar

	BasePagesUrl   string
	IncludesDir    string
	PagesDir       string
	SidebarYmlPath string
}

func NewJekyllPagesGenerator(basePagesUrl, pagesDir, includesDir, sidebarYmlPath string) *JekyllPagesGenerator {
	sidebarFileBasename := filepath.Base(sidebarYmlPath)
	sidebarName := strings.TrimSuffix(sidebarFileBasename, filepath.Ext(sidebarFileBasename))

	return &JekyllPagesGenerator{
		BasePagesUrl:   basePagesUrl,
		PagesDir:       pagesDir,
		IncludesDir:    includesDir,
		SidebarYmlPath: sidebarYmlPath,

		markdownPagesGenerator: NewMarkdownPagesGenerator(includesDir),
		jekyllSidebar:          NewJekyllSidebar(sidebarName, basePagesUrl),
	}
}

func (g *JekyllPagesGenerator) getPagesIncludePrefix() string {
	pagesDirParts := SplitFilepath(g.PagesDir)
	includesDirParts := SplitFilepath(g.IncludesDir)

	var max int
	if len(pagesDirParts) < len(includesDirParts) {
		max = len(pagesDirParts)
	} else {
		max = len(includesDirParts)
	}

	var commonPathParts []string

	for i := 0; i < max; i++ {
		if includesDirParts[len(includesDirParts)-1-i] != pagesDirParts[len(pagesDirParts)-1-i] {
			break
		}
		commonPathParts = append([]string{includesDirParts[len(includesDirParts)-1-i]}, commonPathParts...)
	}

	return filepath.Join(commonPathParts...)
}

func (g *JekyllPagesGenerator) HasFormatPathLink() bool {
	return true
}

func (g *JekyllPagesGenerator) FormatPathLink(markdownPagePath string) string {
	path := path.Join("/", strings.TrimPrefix(g.BasePagesUrl, "/"), strings.TrimSuffix(markdownPagePath, ".md")+".html")
	return fmt.Sprintf("{{ %q | true_relative_url }}", path)
}

func (g *JekyllPagesGenerator) HandlePath(pathPattern string, doc []byte) error {
	// Write markdown partial into includes dir
	if err := g.markdownPagesGenerator.HandlePath(pathPattern, doc); err != nil {
		return fmt.Errorf("unable to generate markdown includes: %s", err)
	}

	markdownPagePath, err := FormatPathPatternAsFilesystemMarkdownPath(pathPattern)
	if err != nil {
		return err
	}

	f := filepath.Join(g.PagesDir, markdownPagePath)
	if err := os.MkdirAll(filepath.Dir(f), os.ModePerm); err != nil {
		return fmt.Errorf("unable to make dir %q: %s", filepath.Dir(f), err)
	}

	pageRelativeUrl := path.Join(strings.TrimPrefix(g.BasePagesUrl, "/"), strings.TrimSuffix(markdownPagePath, ".md")+".html")
	includeRelativePath := path.Join(strings.TrimPrefix(g.getPagesIncludePrefix(), "/"), markdownPagePath)

	var title string
	if pathPattern == "/" {
		title = "Overview"
	} else {
		title = pathPattern
	}

	if err := os.WriteFile(f, []byte(fmt.Sprintf(`---
title: %s
permalink: %s
toc: true
---

{%% include %s %%}
`,
		title,
		pageRelativeUrl,
		path.Join("/", includeRelativePath),
	)), 0644); err != nil {
		return fmt.Errorf("unable to write file %q: %s", f, err)
	}

	if err := g.jekyllSidebar.HandlePath(pathPattern, doc); err != nil {
		return fmt.Errorf("unable to generate sidebar: %s", err)
	}

	return nil
}

func (g *JekyllPagesGenerator) Close() error {
	return g.jekyllSidebar.WriteFile(g.SidebarYmlPath)
}
