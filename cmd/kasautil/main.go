package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	"github.com/cfunkhouser/kasa"
	"github.com/cfunkhouser/kasa/export"
)

// Version of kasautil.
var Version = "development"

var (
	versionMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kasa_exporter_version",
		Help: "Version information about this binary",
		ConstLabels: map[string]string{
			"version": Version,
		},
	})
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

func setState(c *cli.Context, state bool) error {
	daddr, laddr, err := parseAddrs(c)
	if err != nil {
		return cli.Exit(err, 1)
	}
	return kasa.SetRelayState(c.Context, daddr, laddr, state)
}

func serveExporter(c *cli.Context) error {
	var laddr *net.UDPAddr
	if l := c.String("local"); l != "" {
		var err error
		if laddr, err = kasa.ParseAddr(l); err != nil {
			return err
		}
	}
	r := prometheus.NewRegistry()
	r.Register(versionMetric)
	versionMetric.Set(1.0)
	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	http.Handle("/scrape", export.New(export.WithLocalAddr(laddr)))
	return http.ListenAndServe(":9142", nil)
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
		Name:    "kasautil",
		Usage:   "Control Kasa devices on the local network",
		Version: Version,
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List kasa devices on the local network.",
				Flags: append(
					commonFlags,
					&cli.StringFlag{
						Name:    "device",
						Aliases: []string{"d", "discover"},
						Usage:   "Broadcast ip:port target for discovery requests",
						Value:   "255.255.255.255:9999",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Usage:   "Possible values: promsd, human",
						Value:   "human",
					}),
				Action: func(c *cli.Context) error {
					daddr, laddr, err := parseAddrs(c)
					if err != nil {
						return cli.Exit(err, 1)
					}
					format, err := parseFormatter(c)
					if err != nil {
						return cli.Exit(err, 1)
					}
					infos, err := kasa.GetSystemInformation(c.Context, daddr, laddr, false)
					if err != nil {
						return err
					}
					format(os.Stdout, infos)
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
			{
				Name:   "export",
				Usage:  "Export Kasa metrics to Prometheus. Blocks until killed.",
				Flags:  commonFlags,
				Action: serveExporter,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
