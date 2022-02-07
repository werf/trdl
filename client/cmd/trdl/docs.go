package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/client/cmd/trdl/command"
)

func docsCmd(commandGroups *command.Groups) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "docs JEKYLL_SITE_DIR",
		DisableFlagsInUseLine: true,
		Short:                 "Generate documentation as markdown",
		Hidden:                true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			jekyllSiteDir := args[0]

			partialsDir := filepath.Join(jekyllSiteDir, "_includes/documentation/reference/cli")
			pagesDir := filepath.Join(jekyllSiteDir, "pages_en/documentation/reference/cli")
			sidebarPath := filepath.Join(jekyllSiteDir, "_data/sidebars/_cli.yml")

			for _, path := range []string{partialsDir, pagesDir} {
				if err := createEmptyFolder(path); err != nil {
					return err
				}
			}

			if err := command.GenCliPartials(cmd.Root(), partialsDir); err != nil {
				return err
			}

			if err := command.GenCliOverview(*commandGroups, pagesDir); err != nil {
				return err
			}

			if err := command.GenCliPages(*commandGroups, pagesDir); err != nil {
				return err
			}

			if err := command.GenCliSidebar(*commandGroups, sidebarPath); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func createEmptyFolder(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("unable to remove %s: %s", path, err)
	}

	if err := os.MkdirAll(path, 0o777); err != nil {
		return fmt.Errorf("unable to make dir %s: %s", path, err)
	}

	return nil
}
