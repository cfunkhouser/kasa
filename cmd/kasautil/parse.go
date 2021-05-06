package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/cfunkhouser/kasa"
	"github.com/urfave/cli/v2"
)

func parseAddrs(c *cli.Context) (daddr, laddr *net.UDPAddr, err error) {
	daddr, err = kasa.ParseAddr(c.String("device"))
	if err != nil {
		return
	}
	if l := c.String("local"); l != "" {
		if laddr, err = kasa.ParseAddr(l); err != nil {
			return
		}
	}
	return
}

func parseFormatter(c *cli.Context) (formatter, error) {
	f := c.String("format")
	switch f {
	case "promsd":
		return promFileSD, nil
	case "", "human":
		return human, nil
	}
	return human, fmt.Errorf("unsupported format %q", f)
}

func parseOutFile(c *cli.Context) (io.Writer, error) {
	if o := c.String("output"); o != "" {
		return os.Create(o)
	}
	return os.Stdout, nil
}
