package main

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

type sitesCommand struct{}

func (c *sitesCommand) Name() string             { return "sites" }
func (c *sitesCommand) Synopsis() string         { return "Retrieve list of sites" }
func (c *sitesCommand) Usage() string            { return `sites` }
func (c *sitesCommand) SetFlags(f *flag.FlagSet) {}

func (c *sitesCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	_, err := newControllerFromTopFlags(ctx, args[0].(*topFlags))
	if err != nil {
		return handleCommandError(err)
	}

	return subcommands.ExitSuccess
}
