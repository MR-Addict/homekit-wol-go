package config

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brutella/hap"
	"gopkg.in/yaml.v3"
)

const (
	defaultStoragePath  = "./db"
	defaultBroadcastIP  = "255.255.255.255"
	defaultPort         = 9
	defaultName         = "Wake Target"
	defaultManufacturer = "homekit-wol"
	defaultModel        = "wake-switch"
	defaultFirmware     = "1.0.0"
)

type Config struct {
	HomeKit HomeKitConfig `yaml:"homekit"`
	Device  DeviceConfig  `yaml:"device"`
}

type HomeKitConfig struct {
	Name          string   `yaml:"name"`
	Pin           string   `yaml:"pin"`
	StoragePath   string   `yaml:"storage_path"`
	ListenAddress string   `yaml:"listen_address,omitempty"`
	Interfaces    []string `yaml:"interfaces,omitempty"`
	SerialNumber  string   `yaml:"serial_number,omitempty"`
	Manufacturer  string   `yaml:"manufacturer,omitempty"`
	Model         string   `yaml:"model,omitempty"`
	Firmware      string   `yaml:"firmware,omitempty"`
}

type DeviceConfig struct {
	Name        string `yaml:"name"`
	MAC         string `yaml:"mac"`
	BroadcastIP string `yaml:"broadcast_ip,omitempty"`
	Port        int    `yaml:"port,omitempty"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode yaml: %w", err)
	}

	cfg.normalize()
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg *Config) normalize() {
	cfg.HomeKit.Name = strings.TrimSpace(cfg.HomeKit.Name)
	cfg.HomeKit.Pin = normalizePin(cfg.HomeKit.Pin)
	cfg.HomeKit.StoragePath = strings.TrimSpace(cfg.HomeKit.StoragePath)
	if cfg.HomeKit.StoragePath != "" {
		cfg.HomeKit.StoragePath = filepath.Clean(cfg.HomeKit.StoragePath)
	}
	cfg.HomeKit.ListenAddress = strings.TrimSpace(cfg.HomeKit.ListenAddress)
	cfg.HomeKit.SerialNumber = strings.TrimSpace(cfg.HomeKit.SerialNumber)
	cfg.HomeKit.Manufacturer = strings.TrimSpace(cfg.HomeKit.Manufacturer)
	cfg.HomeKit.Model = strings.TrimSpace(cfg.HomeKit.Model)
	cfg.HomeKit.Firmware = strings.TrimSpace(cfg.HomeKit.Firmware)

	for index, iface := range cfg.HomeKit.Interfaces {
		cfg.HomeKit.Interfaces[index] = strings.TrimSpace(iface)
	}

	cfg.Device.Name = strings.TrimSpace(cfg.Device.Name)
	cfg.Device.MAC = strings.TrimSpace(cfg.Device.MAC)
	cfg.Device.BroadcastIP = strings.TrimSpace(cfg.Device.BroadcastIP)

	if hw, err := net.ParseMAC(cfg.Device.MAC); err == nil {
		cfg.Device.MAC = strings.ToLower(hw.String())
		if cfg.HomeKit.SerialNumber == "" {
			cfg.HomeKit.SerialNumber = strings.ToUpper(strings.ReplaceAll(hw.String(), ":", ""))
		}
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.Device.Name == "" && cfg.HomeKit.Name != "" {
		cfg.Device.Name = cfg.HomeKit.Name
	}
	if cfg.HomeKit.Name == "" && cfg.Device.Name != "" {
		cfg.HomeKit.Name = cfg.Device.Name
	}
	if cfg.Device.Name == "" {
		cfg.Device.Name = defaultName
	}
	if cfg.HomeKit.Name == "" {
		cfg.HomeKit.Name = cfg.Device.Name
	}

	if cfg.HomeKit.StoragePath == "" {
		cfg.HomeKit.StoragePath = defaultStoragePath
	}
	if cfg.Device.BroadcastIP == "" {
		cfg.Device.BroadcastIP = defaultBroadcastIP
	}
	if cfg.Device.Port == 0 {
		cfg.Device.Port = defaultPort
	}
	if cfg.HomeKit.Manufacturer == "" {
		cfg.HomeKit.Manufacturer = defaultManufacturer
	}
	if cfg.HomeKit.Model == "" {
		cfg.HomeKit.Model = defaultModel
	}
	if cfg.HomeKit.Firmware == "" {
		cfg.HomeKit.Firmware = defaultFirmware
	}
	if cfg.HomeKit.SerialNumber == "" {
		cfg.HomeKit.SerialNumber = strings.ToUpper(strings.ReplaceAll(cfg.Device.MAC, ":", ""))
	}
}

func (cfg Config) Validate() error {
	var problems []string

	if cfg.HomeKit.Pin == "" {
		problems = append(problems, "homekit.pin is required")
	} else if len(cfg.HomeKit.Pin) != 8 || strings.Trim(cfg.HomeKit.Pin, "0123456789") != "" {
		problems = append(problems, "homekit.pin must be 8 digits or the Apple Home style 3-2-3 format")
	} else if hap.InvalidPins[cfg.HomeKit.Pin] {
		problems = append(problems, "homekit.pin uses a HomeKit-reserved invalid pin")
	}

	if cfg.HomeKit.Name == "" {
		problems = append(problems, "homekit.name is required")
	}
	if cfg.HomeKit.StoragePath == "" {
		problems = append(problems, "homekit.storage_path is required")
	}
	if cfg.HomeKit.SerialNumber == "" {
		problems = append(problems, "homekit.serial_number could not be derived; set device.mac or serial_number")
	}

	if cfg.HomeKit.ListenAddress != "" {
		_, portText, err := net.SplitHostPort(cfg.HomeKit.ListenAddress)
		if err != nil {
			problems = append(problems, "homekit.listen_address must be in host:port form")
		} else if port, err := strconv.Atoi(portText); err != nil || port < 1 || port > 65535 {
			problems = append(problems, "homekit.listen_address must use a valid TCP port")
		}
	}

	for _, iface := range cfg.HomeKit.Interfaces {
		if iface == "" {
			problems = append(problems, "homekit.interfaces cannot contain empty values")
			break
		}
	}

	if cfg.Device.MAC == "" {
		problems = append(problems, "device.mac is required")
	} else if _, err := net.ParseMAC(cfg.Device.MAC); err != nil {
		problems = append(problems, "device.mac must be a valid MAC address")
	}

	if cfg.Device.BroadcastIP == "" {
		problems = append(problems, "device.broadcast_ip is required")
	} else if ip := net.ParseIP(cfg.Device.BroadcastIP); ip == nil || ip.To4() == nil {
		problems = append(problems, "device.broadcast_ip must be a valid IPv4 address")
	}

	if cfg.Device.Port < 1 || cfg.Device.Port > 65535 {
		problems = append(problems, "device.port must be between 1 and 65535")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
	}

	return nil
}

func normalizePin(pin string) string {
	var builder strings.Builder
	builder.Grow(len(pin))
	for _, ch := range strings.TrimSpace(pin) {
		if ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
		}
	}
	return builder.String()
}
