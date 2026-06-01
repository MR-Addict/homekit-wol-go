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
	defaultBridgeName   = "Wake Targets"
	defaultPin          = "00102003"
	defaultStoragePath  = "./db"
	defaultBroadcastIP  = "255.255.255.255"
	defaultPort         = 9
	defaultManufacturer = "homekit-wol"
	defaultModel        = "wake-switch"
	defaultFirmware     = "1.0.0"
)

type Config struct {
	HomeKit HomeKitConfig  `yaml:"homekit"`
	WOL     WOLConfig      `yaml:"wol,omitempty"`
	Devices []DeviceConfig `yaml:"devices"`
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

type WOLConfig struct {
	BroadcastIP string `yaml:"broadcast_ip,omitempty"`
	Port        int    `yaml:"port,omitempty"`
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

	cfg.WOL.BroadcastIP = strings.TrimSpace(cfg.WOL.BroadcastIP)

	for index := range cfg.Devices {
		cfg.Devices[index].Name = strings.TrimSpace(cfg.Devices[index].Name)
		cfg.Devices[index].MAC = strings.TrimSpace(cfg.Devices[index].MAC)
		cfg.Devices[index].BroadcastIP = strings.TrimSpace(cfg.Devices[index].BroadcastIP)

		if hw, err := net.ParseMAC(cfg.Devices[index].MAC); err == nil {
			cfg.Devices[index].MAC = strings.ToLower(hw.String())
		}
	}
}

func (cfg *Config) applyDefaults() {
	if cfg.HomeKit.Name == "" {
		cfg.HomeKit.Name = defaultBridgeName
	}
	if cfg.HomeKit.Pin == "" {
		cfg.HomeKit.Pin = defaultPin
	}

	if cfg.HomeKit.StoragePath == "" {
		cfg.HomeKit.StoragePath = defaultStoragePath
	}
	if cfg.WOL.BroadcastIP == "" {
		cfg.WOL.BroadcastIP = defaultBroadcastIP
	}
	if cfg.WOL.Port == 0 {
		cfg.WOL.Port = defaultPort
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

	for index := range cfg.Devices {
		if cfg.Devices[index].BroadcastIP == "" {
			cfg.Devices[index].BroadcastIP = cfg.WOL.BroadcastIP
		}
		if cfg.Devices[index].Port == 0 {
			cfg.Devices[index].Port = cfg.WOL.Port
		}
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

	if cfg.HomeKit.StoragePath == "" {
		problems = append(problems, "homekit.storage_path is required")
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

	defaultBroadcastIPValid := isValidIPv4(cfg.WOL.BroadcastIP)
	if cfg.WOL.BroadcastIP == "" {
		problems = append(problems, "wol.broadcast_ip is required")
	} else if !defaultBroadcastIPValid {
		problems = append(problems, "wol.broadcast_ip must be a valid IPv4 address")
	}

	defaultPortValid := isValidPort(cfg.WOL.Port)
	if !defaultPortValid {
		problems = append(problems, "wol.port must be between 1 and 65535")
	}

	if len(cfg.Devices) == 0 {
		problems = append(problems, "devices must contain at least one device")
	}

	seenNames := make(map[string]int, len(cfg.Devices))
	seenMACs := make(map[string]int, len(cfg.Devices))
	for index, device := range cfg.Devices {
		path := fmt.Sprintf("devices[%d]", index)

		if device.Name == "" {
			problems = append(problems, path+".name is required")
		} else {
			nameKey := strings.ToLower(device.Name)
			if previousIndex, exists := seenNames[nameKey]; exists {
				problems = append(problems, fmt.Sprintf("%s.name duplicates devices[%d].name", path, previousIndex))
			} else {
				seenNames[nameKey] = index
			}
		}

		if device.MAC == "" {
			problems = append(problems, path+".mac is required")
		} else if hardwareAddr, err := net.ParseMAC(device.MAC); err != nil {
			problems = append(problems, path+".mac must be a valid MAC address")
		} else if len(hardwareAddr) != 6 {
			problems = append(problems, path+".mac must be a 6-byte MAC address")
		} else if previousIndex, exists := seenMACs[device.MAC]; exists {
			problems = append(problems, fmt.Sprintf("%s.mac duplicates devices[%d].mac", path, previousIndex))
		} else {
			seenMACs[device.MAC] = index
		}

		if device.BroadcastIP == "" {
			problems = append(problems, path+".broadcast_ip is required")
		} else if device.BroadcastIP != cfg.WOL.BroadcastIP || defaultBroadcastIPValid {
			if !isValidIPv4(device.BroadcastIP) {
				problems = append(problems, path+".broadcast_ip must be a valid IPv4 address")
			}
		}

		if !isValidPort(device.Port) {
			problems = append(problems, path+".port must be between 1 and 65535")
		}
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

func isValidIPv4(address string) bool {
	ip := net.ParseIP(strings.TrimSpace(address))
	return ip != nil && ip.To4() != nil
}

func isValidPort(port int) bool {
	return port >= 1 && port <= 65535
}
