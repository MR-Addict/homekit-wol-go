package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"

	"homekit-wol/internal/config"
	"homekit-wol/internal/homekit"
	"homekit-wol/internal/wol"
)

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

	info := accessory.Info{
		Name:         cfg.HomeKit.Name,
		SerialNumber: cfg.HomeKit.SerialNumber,
		Manufacturer: cfg.HomeKit.Manufacturer,
		Model:        cfg.HomeKit.Model,
		Firmware:     cfg.HomeKit.Firmware,
	}

	wakeSwitch := homekit.NewWakeSwitch(info, time.Second, func(ctx context.Context) error {
		log.Printf("sending Wake-on-LAN packet to %s (%s)", cfg.Device.Name, cfg.Device.MAC)
		return wol.Send(ctx, cfg.Device.MAC, cfg.Device.BroadcastIP, cfg.Device.Port)
	})

	store := hap.NewFsStore(cfg.HomeKit.StoragePath)
	server, err := hap.NewServer(store, wakeSwitch.A)
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

	log.Printf("HomeKit accessory %q ready for %s", cfg.HomeKit.Name, cfg.Device.Name)
	log.Printf("pair in Apple Home with pin %s", displayPin(cfg.HomeKit.Pin))

	err = server.ListenAndServe(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("serve HomeKit accessory: %w", err)
	}

	return nil
}

func displayPin(pin string) string {
	if len(pin) != 8 {
		return pin
	}

	return fmt.Sprintf("%s-%s-%s", pin[:3], pin[3:5], pin[5:])
}
