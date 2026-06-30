package gendocs

import (
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBackend builds a minimal framework.Backend whose routes carry the given
// synopsis/description pairs, so NewBackendTemplateData can resolve descriptions
// by route path via the public Route API.
func newTestBackend(paths []*framework.Path) *framework.Backend {
	return &framework.Backend{
		Help:        "Test backend help",
		BackendType: 0,
		Paths:       paths,
	}
}

// oasDoc mirrors what Vault exposes at the OpenAPI path level: only the route
// HelpSynopsis ends up in OASPathItem.Description.
func oasDoc(pathSynopsis map[string]string) *framework.OASDocument {
	doc := framework.NewOASDocument("test")
	for path, synopsis := range pathSynopsis {
		doc.Paths[path] = &framework.OASPathItem{
			Description: synopsis,
			Get:         &framework.OASOperation{},
		}
	}
	return doc
}

func pathByName(data *BackendTemplateData, name string) *PathTemplateData {
	for _, p := range data.Paths {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func TestNewBackendTemplateData_UsesHelpDescriptionForHeading(t *testing.T) {
	backend := newTestBackend([]*framework.Path{
		{
			Pattern:         "configure/git_credential/?$",
			HelpSynopsis:    "Configure Git credentials",
			HelpDescription: "Configure Git credentials to perform clone operation",
		},
	})
	doc := oasDoc(map[string]string{
		"/configure/git_credential": "Configure Git credentials",
	})

	data, err := NewBackendTemplateData(doc, backend, nil)
	require.NoError(t, err)

	p := pathByName(data, "/configure/git_credential")
	require.NotNil(t, p)

	// Full-page heading uses HelpDescription.
	assert.Equal(t, "Configure Git credentials to perform clone operation.", p.Description)
	// Overview line keeps the short synopsis (lower-cased).
	assert.Equal(t, "configure git credentials", p.Synopsis)
}

func TestNewBackendTemplateData_FallsBackToSynopsisWhenNoDescription(t *testing.T) {
	backend := newTestBackend([]*framework.Path{
		{
			Pattern:      "configure/pgp_signing_key$",
			HelpSynopsis: "Configure a PGP key for signing release artifacts",
			// No HelpDescription.
		},
	})
	doc := oasDoc(map[string]string{
		"/configure/pgp_signing_key": "Configure a PGP key for signing release artifacts",
	})

	data, err := NewBackendTemplateData(doc, backend, nil)
	require.NoError(t, err)

	p := pathByName(data, "/configure/pgp_signing_key")
	require.NotNil(t, p)

	assert.Equal(t, "Configure a PGP key for signing release artifacts.", p.Description)
	assert.Equal(t, "configure a pgp key for signing release artifacts", p.Synopsis)
}

func TestNewBackendTemplateData_DuplicateSynopsisRoutesGetOwnDescription(t *testing.T) {
	// Two routes deliberately share an identical HelpSynopsis. Matching by route
	// path (not synopsis string) must give each its own HelpDescription.
	backend := newTestBackend([]*framework.Path{
		{
			Pattern:         "alpha$",
			HelpSynopsis:    "Manage thing",
			HelpDescription: "Manage thing alpha in detail",
		},
		{
			Pattern:         "beta$",
			HelpSynopsis:    "Manage thing",
			HelpDescription: "Manage thing beta in detail",
		},
	})
	doc := oasDoc(map[string]string{
		"/alpha": "Manage thing",
		"/beta":  "Manage thing",
	})

	data, err := NewBackendTemplateData(doc, backend, nil)
	require.NoError(t, err)

	alpha := pathByName(data, "/alpha")
	beta := pathByName(data, "/beta")
	require.NotNil(t, alpha)
	require.NotNil(t, beta)

	assert.Equal(t, "Manage thing alpha in detail.", alpha.Description)
	assert.Equal(t, "Manage thing beta in detail.", beta.Description)
}

func TestNewBackendTemplateData_ParametrizedRouteResolves(t *testing.T) {
	backend := newTestBackend([]*framework.Path{
		{
			Pattern:         "configure/trusted_pgp_public_key/" + framework.GenericNameRegex("name") + "$",
			HelpSynopsis:    "Read or delete the configured trusted PGP public key",
			HelpDescription: "Read or delete the configured trusted PGP public key in full",
		},
	})
	doc := oasDoc(map[string]string{
		"/configure/trusted_pgp_public_key/{name}": "Read or delete the configured trusted PGP public key",
	})

	data, err := NewBackendTemplateData(doc, backend, nil)
	require.NoError(t, err)

	p := pathByName(data, "/configure/trusted_pgp_public_key/:name")
	require.NotNil(t, p)

	assert.Equal(t, "Read or delete the configured trusted PGP public key in full.", p.Description)
}

func TestNormalizeHelpString(t *testing.T) {
	cases := map[string]string{
		"  hello   world  ": "hello world",
		"a\n\tb":            "a b",
		"single":            "single",
		"":                  "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeHelpString(in))
	}
}
