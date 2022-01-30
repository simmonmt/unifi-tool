package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"syscall"

	"github.com/google/subcommands"
	"github.com/simmonmt/unifi_tool/lib/unifi"
	"golang.org/x/term"
)

var (
	UsageError = errors.New("usage error")
)

type topFlags struct {
	Username        string
	PasswordEnvName string
	Site            string
	Controller      string
}

func newUsageError(msg string) error {
	return fmt.Errorf("%w: %v", UsageError, msg)
}

func readPassword() (string, error) {
	fmt.Print("Password: ")
	password, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return "", err
	}

	return string(password), nil
}

func newSiteControllerFromTopFlags(ctx context.Context, tf *topFlags) (*unifi.Controller, error) {
	if tf.Site == "" {
		return nil, newUsageError("--site is required")
	}

	return newControllerFromTopFlags(ctx, tf)
}

func newControllerFromTopFlags(ctx context.Context, tf *topFlags) (*unifi.Controller, error) {
	if tf.Controller == "" {
		return nil, newUsageError("--controller is required")
	}
	url, err := url.Parse(tf.Controller)
	if err != nil {
		return nil, newUsageError("bad --controller value")
	}

	if tf.Username == "" {
		return nil, newUsageError("--username is required")
	}

	var password string
	if tf.PasswordEnvName != "" {
		password = os.Getenv(tf.PasswordEnvName)
		if password == "" {
			fmt.Println(os.Environ())
			return nil, fmt.Errorf(
				"no password found in environment variable %v",
				tf.PasswordEnvName)
		}
	} else {
		var err error
		password, err = readPassword()
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}
	}

	c := unifi.NewController(tf.Username, tf.Site, url)
	if err := c.Login(ctx, password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
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
	topFlagSet.StringVar(&topFlags.PasswordEnvName, "password_env", "", "Name of the environment variable that contains the password.")
	topFlagSet.StringVar(&topFlags.Site, "site", "", "Site to use")
	topFlagSet.StringVar(&topFlags.Controller, "controller", "", "URL for controller")

	topFlagSet.Parse(os.Args[1:])

	commander := subcommands.NewCommander(topFlagSet, path.Base(os.Args[0]))

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")

	commander.Register(&forceProvisionCommand{}, "controller operations")
	commander.Register(&listDevicesCommand{}, "controller operations")
	commander.Register(&sitesCommand{}, "controller operations")

	ctx := context.Background()
	os.Exit(int(commander.Execute(ctx, topFlags)))
}
