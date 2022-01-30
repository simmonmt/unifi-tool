package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/google/subcommands"
)

type listDevicesCommand struct {
	verbose bool
}

func (c *listDevicesCommand) Name() string     { return "list_devices" }
func (c *listDevicesCommand) Synopsis() string { return "List Unifi devices" }
func (c *listDevicesCommand) Usage() string    { return `list_devices [-v]` }
func (c *listDevicesCommand) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.verbose, "v", false, "verbose output")
}

func (c *listDevicesCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	ctrl, err := newSiteControllerFromTopFlags(ctx, args[0].(*topFlags))
	if err != nil {
		return handleCommandError(err)
	}

	devices, err := ctrl.Devices(ctx)
	if err != nil {
		return handleCommandError(err)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if c.verbose {
		fmt.Fprintln(w, "Name\tIP\tMAC\tExtra Name\tExtra IP\tExtra MAC")
	} else {
		fmt.Fprintln(w, "Name\tIP\tMAC")
	}

	for _, dev := range devices {
		out := make([]string, 6)
		out[0] = dev.Name
		out[1] = dev.IP.String()
		out[2] = dev.MAC.String()

		if !c.verbose || len(dev.ExtraPorts) == 0 {
			fmt.Fprintln(w, strings.Join(out, "\t"))
			continue
		}

		for i, port := range dev.ExtraPorts {
			if i != 0 {
				out[0], out[1], out[2] = "", "", ""
			}

			out[3] = port.Name
			out[4] = port.IP.String()
			out[5] = port.MAC.String()
			fmt.Fprintln(w, strings.Join(out, "\t"))
		}
	}

	w.Flush()

	return subcommands.ExitSuccess
}
