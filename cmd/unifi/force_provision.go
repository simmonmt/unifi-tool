package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type forceProvisionCommand struct{}

func (c *forceProvisionCommand) Name() string             { return "force_provision" }
func (c *forceProvisionCommand) Synopsis() string         { return "Force provision a device" }
func (c *forceProvisionCommand) Usage() string            { return `force_provision device_name` }
func (c *forceProvisionCommand) SetFlags(f *flag.FlagSet) {}

func (c *forceProvisionCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	// ctrl, err := newControllerFromTopFlags(ctx, args[0].(*topFlags))
	// if err != nil {
	// 	return handleCommandError(err)
	// }

	if len(f.Args()) != 1 {
		return handleCommandError(newUsageError("missing device_name argument"))
	}
	name := f.Args()[0]

	mac, err := nameToMAC(name)
	if err != nil {
		return handleCommandError(fmt.Errorf("failed to get MAC for %v: %w",
			name, err))
	}

	fmt.Println(mac)

	return subcommands.ExitSuccess
}
