package main

import (
	"fmt"
	"sort"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "list",
		Short:                 "List of managed repositories",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			repoConfigurationList := c.GetRepoList()
			var repoNameList []string
			repoConfigurationByName := map[string]*trdlClient.RepoConfiguration{}
			for _, repoConfiguration := range repoConfigurationList {
				repoName := repoConfiguration.Name
				repoNameList = append(repoNameList, repoName)
				repoConfigurationByName[repoName] = repoConfiguration
			}

			sort.Strings(repoNameList)

			tbl := table.New("Name", "URL", "Default Channel")
			for _, repoName := range repoNameList {
				defaultChannel := repoConfigurationByName[repoName].DefaultChannel
				if defaultChannel == "" {
					defaultChannel = trdl.DefaultChannel
				}

				tbl.AddRow(repoName, repoConfigurationByName[repoName].Url, defaultChannel)
			}
			tbl.Print()

			return nil
		},
	}

	return cmd
}
