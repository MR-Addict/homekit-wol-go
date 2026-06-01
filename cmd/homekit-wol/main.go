package main

import (
	"context"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"

	"homekit-wol/internal/config"
	"homekit-wol/internal/homekit"
	"homekit-wol/internal/wol"
)

const bridgeAccessoryID uint64 = 1

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "path to YAML configuration")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := os.MkdirAll(cfg.HomeKit.StoragePath, 0o755); err != nil {
		return fmt.Errorf("create storage path: %w", err)
	}

	accessories, err := buildServerAccessories(cfg)
	if err != nil {
		return fmt.Errorf("build HomeKit accessories: %w", err)
	}

	store := hap.NewFsStore(cfg.HomeKit.StoragePath)
	server, err := hap.NewServer(store, accessories[0], accessories[1:]...)
	if err != nil {
		return fmt.Errorf("create HomeKit server: %w", err)
	}

	server.Pin = cfg.HomeKit.Pin
	if cfg.HomeKit.ListenAddress != "" {
		server.Addr = cfg.HomeKit.ListenAddress
	}
	if len(cfg.HomeKit.Interfaces) > 0 {
		server.Ifaces = cfg.HomeKit.Interfaces
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("HomeKit bridge %q ready with %d devices", cfg.HomeKit.Name, len(cfg.Devices))
	for _, device := range cfg.Devices {
		log.Printf("configured wake target %q (%s) via %s:%d", device.Name, device.MAC, device.BroadcastIP, device.Port)
	}
	log.Printf("pair in Apple Home with pin %s", displayPin(cfg.HomeKit.Pin))

	err = server.ListenAndServe(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("serve HomeKit accessory: %w", err)
	}

	return nil
}

func buildServerAccessories(cfg config.Config) ([]*accessory.A, error) {
	bridge := accessory.NewBridge(accessory.Info{
		Name:         cfg.HomeKit.Name,
		SerialNumber: bridgeSerialNumber(cfg),
		Manufacturer: cfg.HomeKit.Manufacturer,
		Model:        cfg.HomeKit.Model,
		Firmware:     cfg.HomeKit.Firmware,
	})
	bridge.Id = bridgeAccessoryID

	accessories := make([]*accessory.A, 0, len(cfg.Devices)+1)
	accessories = append(accessories, bridge.A)

	for _, device := range cfg.Devices {
		device := device

		wakeSwitch := homekit.NewWakeSwitch(accessory.Info{
			Name:         device.Name,
			SerialNumber: serialNumberForMAC(device.MAC),
			Manufacturer: cfg.HomeKit.Manufacturer,
			Model:        cfg.HomeKit.Model,
			Firmware:     cfg.HomeKit.Firmware,
		}, homekit.DefaultResetDelay, func(ctx context.Context) error {
			log.Printf("sending Wake-on-LAN packet to %s (%s)", device.Name, device.MAC)
			return wol.Send(ctx, device.MAC, device.BroadcastIP, device.Port)
		})

		accessoryID, err := accessoryIDForMAC(device.MAC)
		if err != nil {
			return nil, fmt.Errorf("derive accessory id for %q: %w", device.Name, err)
		}
		wakeSwitch.Id = accessoryID

		accessories = append(accessories, wakeSwitch.A)
	}

	return accessories, nil
}

func bridgeSerialNumber(cfg config.Config) string {
	if cfg.HomeKit.SerialNumber != "" {
		return cfg.HomeKit.SerialNumber
	}

	macs := make([]string, 0, len(cfg.Devices))
	for _, device := range cfg.Devices {
		macs = append(macs, serialNumberForMAC(device.MAC))
	}
	sort.Strings(macs)

	sum := sha1.Sum([]byte(strings.Join(macs, ",")))
	return fmt.Sprintf("BRIDGE-%X", sum[:6])

}

func serialNumberForMAC(mac string) string {
	return strings.ToUpper(strings.ReplaceAll(mac, ":", ""))
}

func accessoryIDForMAC(mac string) (uint64, error) {
	hardwareAddr, err := net.ParseMAC(mac)
	if err != nil {
		return 0, fmt.Errorf("parse mac address: %w", err)
	}
	if len(hardwareAddr) != 6 {
		return 0, fmt.Errorf("mac address must be 6 bytes")
	}

	var id uint64
	for _, octet := range hardwareAddr {
		id = (id << 8) | uint64(octet)
	}

	return id + bridgeAccessoryID + 1, nil
}

func displayPin(pin string) string {
	if len(pin) != 8 {
		return pin
	}

	return fmt.Sprintf("%s-%s-%s", pin[:3], pin[3:5], pin[5:])
}
