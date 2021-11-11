package gendocs

import (
	"context"
	"fmt"
	"math/rand"
	"regexp/syntax"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	regen "github.com/zach-klippenstein/goregen"
)

func GeneratePagesForBackend(ctx context.Context, pagesGenerator PagesGenerator, backend BackendHandle) error {
	pages, err := GetBackendReferencePages(ctx, backend)
	if err != nil {
		return fmt.Errorf("unable to generate backend pages: %s", err)
	}

	return GeneratePages(pagesGenerator, pages)
}

func GeneratePages(pagesGenerator PagesGenerator, pages []*BackendReferencePage) error {
	for _, page := range pages {
		if err := pagesGenerator.HandlePath(page.Path, page.Doc); err != nil {
			return fmt.Errorf("unable to handle path pattern %q: %s", page.Path, err)
		}
	}

	if err := pagesGenerator.Close(); err != nil {
		return fmt.Errorf("unable to close pages generator: %s", err)
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

func GetBackendReferencePages(ctx context.Context, backend BackendHandle) ([]*BackendReferencePage, error) {
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
			return nil, fmt.Errorf("unable to parse openapi docs from backend help response: %s", err)
		}
	default:
		return nil, fmt.Errorf("no openapi backend docs found")
	}

	backendTemplateData, err := NewBackendTemplateData(backendDoc, backend.FrameworkBackendRef)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare backend template data: %s", err)
	}

	overview, err := ExecuteTemplate(BackendOverviewTemplate, backendTemplateData)
	if err != nil {
		return nil, fmt.Errorf("error executing backend overview template: %s", err)
	}

	res = append(res, &BackendReferencePage{
		Path: "/",
		Doc:  []byte(overview),
	})

	for _, pathTemplateData := range backendTemplateData.Paths {
		pathPage, err := ExecuteTemplate(PathTemplate, pathTemplateData)
		if err != nil {
			return nil, fmt.Errorf("error executing path %q template: %s", pathTemplateData.Name, err)
		}

		res = append(res, &BackendReferencePage{
			Path: pathTemplateData.Name,
			Doc:  []byte(pathPage),
		})
	}

	return res, nil
}

func generatorArgs() *regen.GeneratorArgs {
	return &regen.GeneratorArgs{
		RngSource: rand.NewSource(445),
		Flags:     syntax.Perl,
	}
}
