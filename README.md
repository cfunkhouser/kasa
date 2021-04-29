# kasa

A library and command line utility for interacting with tp-link Kasa devices on
your local network.

## Command Line

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