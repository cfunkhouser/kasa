package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/cfunkhouser/kasa"
)

func parseAddr(addr string) (*net.UDPAddr, error) {
	s := strings.Split(addr, ":")
	if len(s) != 2 {
		return nil, fmt.Errorf("not sure what to do with %q, specify ip:port", addr)
	}
	port, err := strconv.Atoi(s[1])
	if err != nil {
		return nil, fmt.Errorf("not sure what to do with port %q, specify ip:port", s[1])
	}
	ip := net.ParseIP(s[0])
	if ip == nil {
		return nil, fmt.Errorf("not sure what to do with IP %q, specify ip:port", s[0])
	}
	return &net.UDPAddr{
		IP:   net.ParseIP(s[0]),
		Port: port,
	}, nil
}

func parseAddrs(c *cli.Context) (daddr, laddr *net.UDPAddr, err error) {
	daddr, err = parseAddr(c.String("device"))
	if err != nil {
		return
	}
	if l := c.String("local"); l != "" {
		if laddr, err = parseAddr(l); err != nil {
			return
		}
	}
	return
}

func setState(c *cli.Context, state bool) error {
	daddr, laddr, err := parseAddrs(c)
	if err != nil {
		return cli.Exit(err, 1)
	}
	return kasa.SetRelayState(c.Context, daddr, laddr, state)
}

var commonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "local",
		Usage:   "Local ip:port from which to send discovery requests",
		Aliases: []string{"L"},
	},
}

func main() {
	app := &cli.App{
		Name:  "kasautil",
		Usage: "Control Kasa devices on the local network",
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List kasa devices on the local network.",
				Flags: append(commonFlags, &cli.StringFlag{
					Name:    "device",
					Aliases: []string{"d", "discover"},
					Usage:   "Broadcast ip:port target for discovery requests",
					Value:   "255.255.255.255:9999",
				}),
				Action: func(c *cli.Context) error {
					daddr, laddr, err := parseAddrs(c)
					if err != nil {
						return cli.Exit(err, 1)
					}
					infos, err := kasa.GetSystemInformation(c.Context, daddr, laddr, false)
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
				Name:  "off",
				Usage: `Set a kasa device to "off"`,
				Flags: append(commonFlags, &cli.StringFlag{
					Name:     "device",
					Aliases:  []string{"d"},
					Required: true,
					Usage:    "ip:port of Kasa device",
				}),
				Action: func(c *cli.Context) error {
					return setState(c, false)
				},
			},
			{
				Name:  "on",
				Usage: `Set a kasa device to "on"`,
				Flags: append(commonFlags, &cli.StringFlag{
					Name:     "device",
					Aliases:  []string{"d"},
					Required: true,
					Usage:    "ip:port of Kasa device",
				}),
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
