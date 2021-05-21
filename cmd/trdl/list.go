package main

import (
	"fmt"
	"sort"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "list",
		Short:                 "List projects",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			projectConfigurationList := c.ListProjects()
			var projectNameList []string
			projectConfigurationByName := map[string]*trdlClient.ProjectConfiguration{}
			for _, projectConfiguration := range projectConfigurationList {
				projectName := projectConfiguration.Name
				projectNameList = append(projectNameList, projectName)
				projectConfigurationByName[projectName] = projectConfiguration
			}

			sort.Strings(projectNameList)

			tbl := table.New("Name", "Repo URL")
			for _, projectName := range projectNameList {
				tbl.AddRow(projectName, projectConfigurationByName[projectName].RepoUrl)
			}
			tbl.Print()

			return nil
		},
	}

	return cmd
}
