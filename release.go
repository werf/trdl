package trdl

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathRelease(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `release$`,
		Fields: map[string]*framework.FieldSchema{
			"git-tag": {
				Type:        framework.TypeString,
				Description: "Project git repository tag which should be released (required)",
			},
			"command": {
				Type:        framework.TypeString,
				Description: "Run specified command in the root of project git repository tag (required)",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathRelease,
				Summary:  pathReleaseHelpSyn,
			},
		},

		HelpSynopsis:    pathReleaseHelpSyn,
		HelpDescription: pathReleaseHelpDesc,
	}
}

func (b *backend) pathRelease(_ context.Context, _ *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	gitTag := d.Get("git-tag").(string)
	if gitTag == "" {
		return logical.ErrorResponse("missing git-tag"), nil
	}

	command := d.Get("command").(string)
	if command == "" {
		return logical.ErrorResponse("missing command"), nil
	}

	fmt.Printf("PERFORMING THE RELEASE! git-tag=%q command=%q\n", gitTag, command)

	return &logical.Response{
		Warnings: []string{"NOT IMPLEMENTED YET"},
	}, nil
}

const (
	pathReleaseHelpSyn = `
	Performs release of project.
	`

	pathReleaseHelpDesc = `
	Performs release of project by the specified git tag.
	Provided command should prepare release artifacts to be published into the /TODO directory.
	`
)
