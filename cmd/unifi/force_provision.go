package main

import (
	"context"
	"flag"
	"fmt"
	"net"

	"github.com/google/subcommands"
	"github.com/simmonmt/unifi_tool/lib/unifi"
)

func findMAC(devices []*unifi.Device, ips []net.IP) net.HardwareAddr {
	for _, ip := range ips {
		for _, device := range devices {
			if ip.Equal(device.IP) {
				return device.MAC
			}

			for _, port := range device.ExtraPorts {
				if ip.Equal(port.IP) {
					return device.MAC
				}
			}
		}
	}

	return nil
}

type forceProvisionCommand struct{}

func (c *forceProvisionCommand) Name() string             { return "force_provision" }
func (c *forceProvisionCommand) Synopsis() string         { return "Force provision a device" }
func (c *forceProvisionCommand) Usage() string            { return `force_provision device_name` }
func (c *forceProvisionCommand) SetFlags(f *flag.FlagSet) {}

func (c *forceProvisionCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		return handleCommandError(newUsageError("missing device_name argument"))
	}
	name := f.Args()[0]

	// Without an available controller we can validate that the user passed
	// a) a MAC address, b) an IP address, or c) a hostname that can be
	// resolved to an IP address. In cases b and c, we can't resolve further
	// to the MAC address using system facilities -- we have to wait until
	// we have a controller.
	//
	// The controller is required for IP->MAC resolution for two
	// reasons. First, retrieving the list of available devices from the
	// controller lets us verify that the MAC we're being asked to
	// reprovision is actually a Unifi device, and is thus eligible for
	// reprovisioning. Second, the MAC address known to this system for the
	// specified IP address may not be the same MAC address known to the
	// controller for that IP address. This happens in particular with the
	// USG, where the MAC address known to the controller is the WAN port's
	// MAC address, and thus won't be present in this system's ARP
	// tables.
	var mac net.HardwareAddr
	ips := []net.IP{}
	if m, err := net.ParseMAC(name); err == nil {
		// It's a MAC address. No further validation needed.
		mac = m
	} else {
		var err error
		if ip := net.ParseIP(name); ip != nil {
			// It's an IP address. We'll need to resolve it later.
			ips = []net.IP{ip}
		} else if ips, err = net.LookupIP(name); err != nil {
			return handleCommandError(fmt.Errorf("failed to resolve %v: %w",
				name, err))
		} else if len(ips) == 0 {
			return handleCommandError(fmt.Errorf("no IPs found for %v", name))
		}
	}

	ctrl, err := newSiteControllerFromTopFlags(ctx, args[0].(*topFlags))
	if err != nil {
		return handleCommandError(err)
	}

	if mac == nil {
		// Look for the MAC address the controller uses for the passed
		// IP address.
		devices, err := ctrl.Devices(ctx)
		if err != nil {
			return handleCommandError(err)
		}

		mac = findMAC(devices, ips)
		if mac == nil {
			return handleCommandError(fmt.Errorf("no MAC found for %v", name))
		}
	} else {
		// In theory we should validate the MAC address against the list
		// of known devices, but if the user is going to go to all the
		// trouble of giving us a MAC address, we might as well trust
		// it. There might be some bug in this code that they're trying
		// to get around.
	}

	if err := ctrl.ForceProvision(ctx, mac); err != nil {
		return handleCommandError(err)
	}

	return subcommands.ExitSuccess
}
