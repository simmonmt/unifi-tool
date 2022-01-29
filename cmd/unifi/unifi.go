package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/google/subcommands"
	"github.com/simmonmt/unifi_tool/lib/unifi"
)

var (
	UsageError = errors.New("usage error")
)

type topFlags struct {
	Username string
	Site     string
}

func newUsageError(msg string) error {
	return fmt.Errorf("%w: %v", UsageError, msg)
}

func readPassword() (string, error) {
	return "", nil
}

func newSiteControllerFromTopFlags(ctx context.Context, tf *topFlags) (*unifi.Controller, error) {
	if tf.Site == "" {
		return nil, newUsageError("--site is required")
	}

	return newControllerFromTopFlags(ctx, tf)
}

func newControllerFromTopFlags(ctx context.Context, tf *topFlags) (*unifi.Controller, error) {
	if tf.Username == "" {
		return nil, newUsageError("--username is required")
	}

	password, err := readPassword()
	if err != nil {
		return nil, err
	}

	c := unifi.NewController(tf.Username, tf.Site)
	if err := c.Login(password); err != nil {
		return nil, err
	}

	return c, nil
}

func handleCommandError(err error) subcommands.ExitStatus {
	fmt.Fprintln(os.Stderr, err)
	if errors.Is(err, UsageError) {
		return subcommands.ExitUsageError
	}
	return subcommands.ExitFailure
}

func main() {
	topFlags := &topFlags{}

	topFlagSet := flag.NewFlagSet("", flag.ExitOnError)
	topFlagSet.StringVar(&topFlags.Username, "username", "", "Username for login. Password will be read interactively.")
	topFlagSet.StringVar(&topFlags.Site, "site", "", "Site to use")

	topFlagSet.Parse(os.Args[1:])

	commander := subcommands.NewCommander(topFlagSet, path.Base(os.Args[0]))

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")
	commander.Register(&sitesCommand{}, "controller operations")

	ctx := context.Background()
	os.Exit(int(commander.Execute(ctx, topFlags)))
}
