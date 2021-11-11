package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/spf13/cobra"

	trdl "github.com/werf/trdl/server"
	"github.com/werf/trdl/server/pkg/gendocs"
)

func NewCmdGenerateJekyll() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate-jekyll",
		Short:   "Generate plugin path help as jekyll pages",
		Example: "vault-plugin-docs generate-jekyll",
		RunE: func(cmd *cobra.Command, args []string) error {
			if generateJekyllData.BasePagesUrl == "" {
				return fmt.Errorf("--base-pages-url required")
			}
			if generateJekyllData.PagesDir == "" {
				return fmt.Errorf("--pages-dir required")
			}
			if generateJekyllData.IncludesDir == "" {
				return fmt.Errorf("--includes-dir required")
			}
			if generateJekyllData.SidebarYmlPath == "" {
				return fmt.Errorf("--sidebar-yml-path required")
			}

			return generateJekyll(context.Background())
		},
	}

	cmd.Flags().StringVarP(&generateJekyllData.BasePagesUrl, "base-pages-url", "P", "", "Url prefix for generated pages (e.g. /reference/vault_plugin)")
	cmd.Flags().StringVarP(&generateJekyllData.PagesDir, "pages-dir", "p", "", "Path to the directory where jekyll pages will be generated (e.g. docs/pages/reference/vault_plugin)")
	cmd.Flags().StringVarP(&generateJekyllData.IncludesDir, "includes-dir", "i", "", "Path to the directory where jekyll include partials will be generated (e.g. docs/_includes/reference/vault_plugin)")
	cmd.Flags().StringVarP(&generateJekyllData.SidebarYmlPath, "sidebar-yml-path", "s", "", "Path to the sidebar yaml file (e.g. docs/_data/sidebars/_vault_plugin.yml)")

	return cmd
}

var generateJekyllData struct {
	BasePagesUrl   string
	PagesDir       string
	IncludesDir    string
	SidebarYmlPath string
}

func generateJekyll(ctx context.Context) error {
	backendHandle, err := getTrdlBackendGendocsHandle(ctx)
	if err != nil {
		return err
	}

	for _, dir := range []string{generateJekyllData.PagesDir, generateJekyllData.IncludesDir, generateJekyllData.SidebarYmlPath} {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to clean %q: %s", dir, err)
		}
	}

	return gendocs.GeneratePagesForBackend(ctx, gendocs.NewJekyllPagesGenerator(generateJekyllData.BasePagesUrl, generateJekyllData.PagesDir, generateJekyllData.IncludesDir, generateJekyllData.SidebarYmlPath), backendHandle)
}

var generateMarkdownData struct {
	Dir string
}

func NewCmdGenerateMarkdown() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate-markdown",
		Short:   "Generate plugin path help as markdown pages",
		Example: "vault-plugin-docs generate-markdown",
		RunE: func(cmd *cobra.Command, args []string) error {
			if generateMarkdownData.Dir == "" {
				return fmt.Errorf("--dir required")
			}

			return generateMarkdown(context.Background())
		},
	}

	cmd.Flags().StringVarP(&generateMarkdownData.Dir, "dir", "d", "", "Target directory to generate markdown files into (e.g. docs/)")

	return cmd
}

func generateMarkdown(ctx context.Context) error {
	backendHandle, err := getTrdlBackendGendocsHandle(ctx)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(generateMarkdownData.Dir); err != nil {
		return fmt.Errorf("unable to clean %q: %s", generateMarkdownData.Dir, err)
	}

	return gendocs.GeneratePagesForBackend(ctx, gendocs.NewMarkdownPagesGenerator(generateMarkdownData.Dir), backendHandle)
}

func getTrdlBackendGendocsHandle(ctx context.Context) (gendocs.BackendHandle, error) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: time.Hour * 12,
			MaxLeaseTTLVal:     time.Hour * 24,
		},
		StorageView: &logical.InmemStorage{},
	}

	b, err := trdl.NewBackend(config.Logger)
	if err != nil {
		return gendocs.BackendHandle{}, err
	}

	if err := b.Setup(ctx, config); err != nil {
		return gendocs.BackendHandle{}, err
	}

	return gendocs.NewBackendHandle(b, b.Backend, config.StorageView), nil
}

func main() {
	rootCmd := &cobra.Command{
		Use: "vault-plugin-docs",
	}

	rootCmd.AddCommand(NewCmdGenerateMarkdown(), NewCmdGenerateJekyll())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
