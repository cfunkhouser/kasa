package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/cfunkhouser/kasa"
)

var ErrNoDevices = errors.New("no devices detected")

func setState(c *cli.Context, state bool) error {
	addr := c.Args().First()
	if addr == "" {
		return cli.Exit("address is required", 1)
	}
	raddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	return kasa.SetRelayState(c.Context, raddr, state)
}

func main() {
	app := &cli.App{
		Name:  "kasautil",
		Usage: "Control Kasa devices on the local network",
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "List kasa devices on the local network.",
				Action: func(c *cli.Context) error {
					infos, err := kasa.Discover(c.Context)
					if err != nil {
						return err
					}
					if len(infos) == 0 {
						fmt.Println("No devices detected on local network")
						return nil
					}
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.DiscardEmptyColumns)
					fmt.Fprintln(w, "Address\tAlias\tState")
					for _, info := range infos {
						state := "Off"
						if info.RelayState == 1 {
							state = "On"
						}
						fmt.Fprintf(w, "%v\t%v\t%v\n", info.RemoteAddress, info.Alias, state)
					}
					w.Flush()
					return nil
				},
			},
			{
				Name:      "off",
				Usage:     `Set a kasa device to "off"`,
				ArgsUsage: "[address]",
				Action: func(c *cli.Context) error {
					return setState(c, false)
				},
			},
			{
				Name:      "on",
				Usage:     `Set a kasa device to "on"`,
				ArgsUsage: "[address]",
				Action: func(c *cli.Context) error {
					return setState(c, true)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
