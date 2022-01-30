package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/subcommands"
)

type sitesCommand struct{}

func (c *sitesCommand) Name() string             { return "sites" }
func (c *sitesCommand) Synopsis() string         { return "Retrieve list of sites" }
func (c *sitesCommand) Usage() string            { return `sites` }
func (c *sitesCommand) SetFlags(f *flag.FlagSet) {}

func (c *sitesCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	ctrl, err := newControllerFromTopFlags(ctx, args[0].(*topFlags))
	if err != nil {
		return handleCommandError(err)
	}

	sites, err := ctrl.Sites(ctx)
	if err != nil {
		return handleCommandError(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Name\tDescription")

	for _, site := range sites {
		fmt.Fprintf(w, "%v\t%v\n", site.Name, site.Desc)
	}
	w.Flush()

	return subcommands.ExitSuccess
}
