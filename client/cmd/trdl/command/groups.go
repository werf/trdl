package command

import (
	"github.com/spf13/cobra"
)

type Group struct {
	Message  string
	Commands []*cobra.Command
}

type Groups []Group

func (g Groups) Add(c *cobra.Command) {
	for _, group := range g {
		c.AddCommand(group.Commands...)
	}
}

func (g Groups) Has(c *cobra.Command) bool {
	for _, group := range g {
		for _, command := range group.Commands {
			if command == c {
				return true
			}
		}
	}
	return false
}
