# kasa

A library and command line utility for interacting with tp-link Kasa devices on
your local network.

## Command Line

Full `kasautil` options:

```console
$ kasautil
NAME:
   kasautil - Control Kasa devices on the local network

USAGE:
   kasautil [global options] command [command options] [arguments...]

COMMANDS:
   list, ls  List kasa devices on the local network.
   off       Set a kasa device to "off"
   on        Set a kasa device to "on"
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)

```

You may discover Kasa devices locally with the `list` command of `kasautil`.

```console
$ kasautil list
Address          Alias             State
10.24.6.14:9999  ADSL Modem        On
10.23.6.15:9999  Living Room Lamp  On
```

The only currently-supported control function is setting the relay state on
supported devices. To do this, provide the Kasa device's address to the `on` or
`off` commands of `kasautil`.

```console
$ kasautil off 10.24.6.14:9999
$ kasautil list
Address          Alias             State
10.24.6.14:9999  ADSL Modem        Off
10.23.6.15:9999  Living Room Lamp  On
```

### Broadcast Issues

Discovery relies on UDP packets sent to a broadcast address. This can fail when
the host on which `kasautil` has many network interfaces or multiple network
addresses on a single interface. This is known to happen on Linux with a single
interface on multiple VLANs, for example.

In this case, you have two options:

1. Specify a local address using `-L` / `--local` on the correct network from
   which to send the Kasa request

2. Specify a device or broadcast address using `-d` / `--device` / `--discover`
   on the correct network to which the Kasa request should be sent

```console
$ kasautil list -L 10.24.6.15:54321
Address          Alias             State
10.24.6.14:9999  ADSL Modem        On
10.23.6.15:9999  Living Room Lamp  On
$ kasautil list -d 10.24.6.255:9999
Address          Alias             State
10.24.6.14:9999  ADSL Modem        On
10.23.6.15:9999  Living Room Lamp  On
```