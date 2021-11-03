package gendocs

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"regexp/syntax"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
	regen "github.com/zach-klippenstein/goregen"
)

func GeneratePagesForBackend(ctx context.Context, pagesGenerator PagesGenerator, backend logical.Backend, storage logical.Storage) error {
	pages, err := GetBackendReferencePages(ctx, backend, storage)
	if err != nil {
		return fmt.Errorf("unable to generate backend pages: %s", err)
	}

	return GeneratePages(pagesGenerator, pages)
}

func GeneratePages(pagesGenerator PagesGenerator, pages []*BackendReferencePage) error {
	for _, page := range pages {
		if err := pagesGenerator.HandlePath(page.PathPattern, page.Doc); err != nil {
			return fmt.Errorf("unable to handle path pattern %q: %s", page.PathPattern, err)
		}
	}

	if err := pagesGenerator.Close(); err != nil {
		return fmt.Errorf("unable to close pages generator: %s", err)
	}

	return nil
}

type BackendReferencePage struct {
	PathPattern string
	Doc         []byte
}

func GetBackendReferencePages(ctx context.Context, backend logical.Backend, storage logical.Storage) ([]*BackendReferencePage, error) {
	req := &logical.Request{
		Operation:  logical.HelpOperation,
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := backend.HandleRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var res []*BackendReferencePage

	overview := resp.Data["help"].(string)

	res = append(res, &BackendReferencePage{
		PathPattern: "/",
		Doc:         []byte(overview),
	})

	scanner := bufio.NewScanner(strings.NewReader(overview))
	var patterns []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "    ^") {
			pattern := strings.TrimPrefix(line, "    ")
			patterns = append(patterns, pattern)
		}
	}

	for _, pattern := range patterns {
		generator, err := regen.NewGenerator(pattern, generatorArgs())
		if err != nil {
			return nil, fmt.Errorf("unable to create regen generator based on pattern %s: %s", pattern, err)
		}

		path := generator.Generate()

		req := &logical.Request{
			Operation:  logical.HelpOperation,
			Path:       path,
			Storage:    storage,
			Connection: &logical.Connection{},
		}

		resp, err := backend.HandleRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		pathHelpRaw := resp.Data["help"].(string)
		scanner := bufio.NewScanner(strings.NewReader(pathHelpRaw))

		var resPathHelpLines []string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Request:") || strings.HasPrefix(line, "Matching Route:") {
				continue
			}
			resPathHelpLines = append(resPathHelpLines, line)
		}

		resPathHelp := fmt.Sprintf("## PATH PATTERN\n\n    %s\n\n%s", pattern, strings.TrimSpace(strings.Join(resPathHelpLines, "\n")))

		pathPattern := strings.TrimSuffix(strings.TrimPrefix(pattern, "^"), "$")

		res = append(res, &BackendReferencePage{
			PathPattern: pathPattern,
			Doc:         []byte(resPathHelp),
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
