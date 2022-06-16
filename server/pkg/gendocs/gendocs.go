package gendocs

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func GeneratePagesForBackend(ctx context.Context, pagesGenerator PagesGenerator, backend BackendHandle) error {
	var formatPathLink func(string) string
	if pagesGenerator.HasFormatPathLink() {
		formatPathLink = pagesGenerator.FormatPathLink
	}

	pages, err := GetBackendReferencePages(ctx, backend, formatPathLink)
	if err != nil {
		return fmt.Errorf("unable to generate backend pages: %w", err)
	}

	return GeneratePages(pagesGenerator, pages)
}

func GeneratePages(pagesGenerator PagesGenerator, pages []*BackendReferencePage) error {
	for _, page := range pages {
		if err := pagesGenerator.HandlePath(page.Path, page.Doc); err != nil {
			return fmt.Errorf("unable to handle path pattern %q: %w", page.Path, err)
		}
	}

	if err := pagesGenerator.Close(); err != nil {
		return fmt.Errorf("unable to close pages generator: %w", err)
	}

	return nil
}

type BackendReferencePage struct {
	Path string
	Doc  []byte
}

type BackendHandle struct {
	LogicalBackendRef   logical.Backend
	FrameworkBackendRef *framework.Backend
	Storage             logical.Storage
}

func NewBackendHandle(logicalBackendRef logical.Backend, frameworkBackendRef *framework.Backend, storage logical.Storage) BackendHandle {
	return BackendHandle{LogicalBackendRef: logicalBackendRef, FrameworkBackendRef: frameworkBackendRef, Storage: storage}
}

func GetBackendReferencePages(ctx context.Context, backend BackendHandle, formatPathLink func(markdownPagePath string) string) ([]*BackendReferencePage, error) {
	req := &logical.Request{
		Operation:  logical.HelpOperation,
		Storage:    backend.Storage,
		Connection: &logical.Connection{},
	}

	resp, err := backend.LogicalBackendRef.HandleRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var res []*BackendReferencePage

	var backendDoc *framework.OASDocument
	switch v := resp.Data["openapi"].(type) {
	case *framework.OASDocument:
		backendDoc = v
	case map[string]interface{}:
		backendDoc, err = framework.NewOASDocumentFromMap(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse openapi docs from backend help response: %w", err)
		}
	default:
		return nil, fmt.Errorf("no openapi backend docs found")
	}

	backendTemplateData, err := NewBackendTemplateData(backendDoc, backend.FrameworkBackendRef, formatPathLink)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare backend template data: %w", err)
	}

	overview, err := ExecuteTemplate(BackendOverviewTemplate, backendTemplateData)
	if err != nil {
		return nil, fmt.Errorf("error executing backend overview template: %w", err)
	}

	res = append(res, &BackendReferencePage{
		Path: "/",
		Doc:  []byte(overview),
	})

	for _, pathTemplateData := range backendTemplateData.Paths {
		pathPage, err := ExecuteTemplate(PathTemplate, pathTemplateData)
		if err != nil {
			return nil, fmt.Errorf("error executing path %q template: %w", pathTemplateData.Name, err)
		}

		res = append(res, &BackendReferencePage{
			Path: pathTemplateData.Name,
			Doc:  []byte(pathPage),
		})
	}

	return res, nil
}
