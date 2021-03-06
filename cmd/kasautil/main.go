package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	"github.com/cfunkhouser/kasa"
	"github.com/cfunkhouser/kasa/export"
)

var (
	// Version of kasautil. Set at build time to something meaningful.
	Version = "development"

	versionMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kasa_exporter_version",
		Help: "Version information about this binary",
		ConstLabels: map[string]string{
			"version": Version,
		},
	})

	defaultCycleSleep         = time.Second * 15
	defaultPromMetricsAddress = ":9142"
)

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
	if err := r.Register(versionMetric); err != nil {
		return err
	}
	versionMetric.Set(1.0)
	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	http.Handle("/scrape", export.New(export.WithLocalAddr(laddr)))
	return http.ListenAndServe(c.String("metricsaddress"), nil)
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
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "File to which output is written. If unset, use STDOUT.",
					},
				),
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
					// Parse and open the output file _after_ the network call,
					// so that if it fails, we don't truncate an extant file with
					// garbage.
					out, err := parseOutFile(c)
					if err != nil {
						return cli.Exit(err, 1)
					}
					format(out, infos)
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
				Name:  "cycle",
				Usage: `Turn a kasa device "off" and then "on." Will end by setting "on" regardless of starting state.`,
				Flags: append(commonFlags,
					&cli.StringFlag{
						Name:     "device",
						Aliases:  []string{"d"},
						Required: true,
						Usage:    "ip:port of Kasa device",
					},
					&cli.DurationFlag{
						Name:    "sleep",
						Aliases: []string{"s"},
						Value:   defaultCycleSleep,
						Usage:   `Time to wait between setting device "off" and "on"`,
					}),
				Action: func(c *cli.Context) error {
					if err := setState(c, false); err != nil {
						return err
					}
					time.Sleep(c.Duration("sleep"))
					return setState(c, true)
				},
			},
			{
				Name:  "export",
				Usage: "Export Kasa metrics to Prometheus. Blocks until killed.",
				Flags: append(commonFlags, &cli.StringFlag{
					Name:    "metricsaddress",
					Aliases: []string{"a"},
					Value:   defaultPromMetricsAddress,
					Usage:   "ip:port from which to serve Prometheus metrics",
				}),
				Action: serveExporter,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
