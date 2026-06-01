# homekit-wol

Minimal Go HomeKit bridge that wakes one or more devices with Wake-on-LAN magic packets. Each configured device appears in Apple Home as its own switch. Turning a switch on sends the packet for that target and the switch resets back to off automatically.

## Requirements

- Go 1.21 or newer
- A device on your LAN that can stay online, such as a Linux host or Raspberry Pi
- Wake-on-LAN enabled on the target device BIOS and network adapter

## Configuration

Start from `configs/config.example.yaml`, then edit `config.yaml`.

```yaml
homekit:
  name: "Wake Targets"

devices:
  - name: "Gaming PC"
    mac: "00:11:22:33:44:55"
  - name: "NAS"
    mac: "66:77:88:99:aa:bb"
```

Notes:

- `homekit.pin` is optional and defaults to `001-02-003`. If you set it explicitly, it can be either `00102003` or `001-02-003`.
- `homekit.name` is the bridge name and defaults to `Wake Targets`.
- `storage_path` defaults to `./db` and is where HomeKit pairing data is kept.
- `wol.broadcast_ip` and `wol.port` provide shared defaults for every device. Each device can override either value individually.
- `broadcast_ip` defaults to `255.255.255.255`, but some networks require the subnet broadcast address instead, for example `192.168.1.255`.
- Device MACs must be standard 6-byte Ethernet MAC addresses.
- Upgrading from the old single-device config requires replacing `device:` with `devices:`. Existing installs may also need the old accessory removed from Apple Home and the `db/` directory cleared before re-pairing the new bridge layout.

## Run

```sh
go test ./...
go run ./cmd/homekit-wol
```

The service logs the HomeKit pin on startup. Pair the accessory in Apple Home with that pin.

When paired, Apple Home shows one bridge plus one switch per configured device.

## Build

Build for the current platform:

```sh
mkdir -p bin
go build -o bin/homekit-wol ./cmd/homekit-wol # Unix
go build -o bin/homekit-wol.exe ./cmd/homekit-wol # Windows
```

Build for Linux on MIPS with softfloat (e.g. Raspberry Pi Zero):

```sh
env GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bin/homekit-wol ./cmd/homekit-wol # Linux MIPS
$env:GOOS="linux"; $env:GOARCH="mipsle"; $env:GOMIPS="softfloat"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -trimpath -o bin/homekit-wol ./cmd/homekit-wol # Linux MIPS
```

Build for Linux on ARM64 (e.g. Raspberry Pi 3/4):

```sh
env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bin/homekit-wol ./cmd/homekit-wol # Linux ARM64
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -trimpath -o bin/homekit-wol ./cmd/homekit-wol # Linux ARM64
```

## Linux service

Example `systemd` unit:

```ini
[Unit]
Description=HomeKit Wake-on-LAN bridge
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/opt/homekit-wol
ExecStart=/opt/homekit-wol/bin/homekit-wol -config /opt/homekit-wol/config.yaml
Restart=on-failure
User=pi

[Install]
WantedBy=multi-user.target
```

Example OpenWRT init script:

```ini
#!/bin/sh /etc/rc.common

USE_PROCD=1
START=99
STOP=10

start_service() {
  procd_open_instance

  # Use /bin/sh to change the directory, then 'exec' the application
  procd_set_param command /bin/sh -c "cd /root/projects/homekit-wol && exec ./homekit-wol"

  # Set the working directory
  procd_set_param env PWD=/root/projects/homekit-wol

  # Forward output to the system log (viewable with 'logread')
  procd_set_param stdout 1
  procd_set_param stderr 1

  # Automatically restart the app if it crashes
  procd_set_param respawn

  procd_close_instance
}
```

## Behavior

- Apple Home shows the accessory as a switch.
- Apple Home shows one switch per configured device under the bridge.
- Turning a switch on sends one magic packet to that device.
- After a short delay, each switch resets to off so it behaves like a trigger.
- Pairing data persists in `db/`, so you do not need to re-pair on every restart.
