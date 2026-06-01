# homekit-wol

Minimal Go HomeKit accessory that wakes one device with a Wake-on-LAN magic packet. It appears in Apple Home as a normal switch. Turning the switch on sends the packet and the switch resets back to off automatically.

## Requirements

- Go 1.21 or newer
- A device on your LAN that can stay online, such as a Linux host or Raspberry Pi
- Wake-on-LAN enabled on the target device BIOS and network adapter

## Configuration

Edit `config.yaml`.

```yaml
homekit:
  pin: "001-02-003"
  name: "Gaming PC"
  storage_path: "./db"

device:
  name: "Gaming PC"
  mac: "00:11:22:33:44:55"
  broadcast_ip: "255.255.255.255"
  port: 9
```

Notes:

- `pin` can be either `00102003` or `001-02-003`.
- `broadcast_ip` defaults to `255.255.255.255`, but some networks require the subnet broadcast address instead, for example `192.168.1.255`.
- `storage_path` is where HomeKit pairing data is kept.

## Run

```sh
go test ./...
go run .
```

The service logs the HomeKit pin on startup. Pair the accessory in Apple Home with that pin.

## Build

Build for the current platform:

```sh
mkdir -p bin
go build -o bin/homekit-wol . # Unix
go build -o bin/homekit-wol.exe . # Windows
```

Build for Linux on MIPS with softfloat (e.g. Raspberry Pi Zero):

```sh
env GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bin/homekit-wol . # Linux MIPS
$env:GOOS="linux"; $env:GOARCH="mipsle"; $env:GOMIPS="softfloat"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -trimpath -o bin/homekit-wol . # Linux MIPS
```

Build for Linux on ARM64 (e.g. Raspberry Pi 3/4):

```sh
env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bin/homekit-wol . # Linux ARM64
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -trimpath -o bin/homekit-wol . # Linux ARM64
```

## Linux service

Example `systemd` unit:

```ini
[Unit]
Description=HomeKit Wake-on-LAN accessory
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

## Behavior

- Apple Home shows the accessory as a switch.
- Turning the switch on sends one magic packet.
- After a short delay, the switch resets to off so it behaves like a trigger.
- Pairing data persists in `db/`, so you do not need to re-pair on every restart.
