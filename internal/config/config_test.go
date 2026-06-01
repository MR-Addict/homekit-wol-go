package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndNormalizesValues(t *testing.T) {
	path := writeTempConfig(t, `
homekit:
  pin: "001-02-003"
device:
  name: "Gaming PC"
  mac: "00-11-22-33-44-55"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HomeKit.Pin != "00102003" {
		t.Fatalf("expected normalized pin, got %q", cfg.HomeKit.Pin)
	}
	if cfg.HomeKit.Name != "Gaming PC" {
		t.Fatalf("expected HomeKit name to default from device name, got %q", cfg.HomeKit.Name)
	}
	if cfg.HomeKit.StoragePath != "./db" {
		t.Fatalf("expected default storage path, got %q", cfg.HomeKit.StoragePath)
	}
	if cfg.HomeKit.SerialNumber != "001122334455" {
		t.Fatalf("expected serial number derived from MAC, got %q", cfg.HomeKit.SerialNumber)
	}
	if cfg.Device.MAC != "00:11:22:33:44:55" {
		t.Fatalf("expected canonical MAC format, got %q", cfg.Device.MAC)
	}
	if cfg.Device.BroadcastIP != "255.255.255.255" {
		t.Fatalf("expected default broadcast IP, got %q", cfg.Device.BroadcastIP)
	}
	if cfg.Device.Port != 9 {
		t.Fatalf("expected default port, got %d", cfg.Device.Port)
	}
}

func TestLoadRejectsInvalidPin(t *testing.T) {
	path := writeTempConfig(t, `
homekit:
  pin: "123-45-678"
device:
  mac: "00:11:22:33:44:55"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected invalid pin to fail validation")
	}
	if !strings.Contains(err.Error(), "homekit.pin") {
		t.Fatalf("expected homekit.pin validation error, got %v", err)
	}
}

func TestLoadRejectsInvalidMAC(t *testing.T) {
	path := writeTempConfig(t, `
homekit:
  pin: "001-02-003"
device:
  mac: "not-a-mac"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected invalid MAC to fail validation")
	}
	if !strings.Contains(err.Error(), "device.mac") {
		t.Fatalf("expected device.mac validation error, got %v", err)
	}
}

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}
