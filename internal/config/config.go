package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultHostIP              = "192.168.10.3"
	DefaultBoxIP               = "192.168.10.2"
	DefaultNetmask             = "255.255.255.0"
	DefaultHardwarePortPattern = "AX88179A"
	DefaultLogLevel            = "DEBUG"
	DefaultListenAddr          = "127.0.0.1:18888"
	DefaultConfigDirName       = ".box-link"
	DefaultConfigFileName      = "config.toml"
)

type Config struct {
	ConfigPath          string
	Iface               string
	HostIP              string
	BoxIP               string
	Netmask             string
	HardwarePortPattern string
	LogLevel            string
	ListenAddr          string
}

func Load() (Config, error) {
	configPath, err := resolveConfigPath()
	if err != nil {
		return Config{}, err
	}

	cfg := defaults(configPath)
	if err := ensureFile(cfg); err != nil {
		return Config{}, err
	}

	fileCfg, err := loadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	mergeConfig(&cfg, fileCfg)
	cfg.ConfigPath = configPath
	if err := syncPathField(cfg, fileCfg); err != nil {
		return Config{}, err
	}
	applyEnvOverrides(&cfg)
	cfg.LogLevel = strings.ToUpper(strings.TrimSpace(cfg.LogLevel))
	cfg.ConfigPath = configPath

	return cfg, nil
}

func defaults(configPath string) Config {
	return Config{
		ConfigPath:          configPath,
		Iface:               "",
		HostIP:              DefaultHostIP,
		BoxIP:               DefaultBoxIP,
		Netmask:             DefaultNetmask,
		HardwarePortPattern: DefaultHardwarePortPattern,
		LogLevel:            DefaultLogLevel,
		ListenAddr:          DefaultListenAddr,
	}
}

func resolveConfigPath() (string, error) {
	if value := strings.TrimSpace(os.Getenv("BOX_CONFIG_PATH")); value != "" {
		return filepath.Abs(value)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, DefaultConfigDirName, DefaultConfigFileName), nil
}

func ensureFile(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if _, err := os.Stat(cfg.ConfigPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file: %w", err)
	}

	if err := os.WriteFile(cfg.ConfigPath, []byte(render(cfg)), 0o644); err != nil {
		return fmt.Errorf("write default config file: %w", err)
	}

	return nil
}

func syncPathField(cfg Config, fileCfg Config) error {
	if strings.TrimSpace(fileCfg.ConfigPath) == cfg.ConfigPath {
		return nil
	}

	fileCfg.ConfigPath = cfg.ConfigPath
	fileCfg.LogLevel = strings.ToUpper(strings.TrimSpace(fileCfg.LogLevel))
	if fileCfg.LogLevel == "" {
		fileCfg.LogLevel = cfg.LogLevel
	}
	mergeConfig(&cfg, fileCfg)

	if err := os.WriteFile(cfg.ConfigPath, []byte(render(cfg)), 0o644); err != nil {
		return fmt.Errorf("update config file metadata: %w", err)
	}

	return nil
}

func loadFile(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	cfg := Config{}
	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("parse config file %s:%d: expected key = value", path, lineNo)
		}

		key = strings.TrimSpace(key)
		value, err := parseValue(rawValue)
		if err != nil {
			return Config{}, fmt.Errorf("parse config file %s:%d: %w", path, lineNo, err)
		}

		switch key {
		case "config_path":
			cfg.ConfigPath = value
		case "iface":
			cfg.Iface = value
		case "host_ip":
			cfg.HostIP = value
		case "box_ip":
			cfg.BoxIP = value
		case "netmask":
			cfg.Netmask = value
		case "hardware_port_pattern":
			cfg.HardwarePortPattern = value
		case "log_level":
			cfg.LogLevel = value
		case "listen_addr":
			cfg.ListenAddr = value
		default:
			return Config{}, fmt.Errorf("parse config file %s:%d: unknown key %q", path, lineNo, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	return cfg, nil
}

func parseValue(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if idx := strings.Index(value, "#"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "\"") {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(unquoted), nil
	}
	return strings.TrimSpace(value), nil
}

func render(cfg Config) string {
	lines := []string{
		"# box-link configuration",
		fmt.Sprintf("config_path = %s", strconv.Quote(cfg.ConfigPath)),
		fmt.Sprintf("iface = %s", strconv.Quote(cfg.Iface)),
		fmt.Sprintf("host_ip = %s", strconv.Quote(cfg.HostIP)),
		fmt.Sprintf("box_ip = %s", strconv.Quote(cfg.BoxIP)),
		fmt.Sprintf("netmask = %s", strconv.Quote(cfg.Netmask)),
		fmt.Sprintf("hardware_port_pattern = %s", strconv.Quote(cfg.HardwarePortPattern)),
		fmt.Sprintf("log_level = %s", strconv.Quote(cfg.LogLevel)),
		fmt.Sprintf("listen_addr = %s", strconv.Quote(cfg.ListenAddr)),
	}

	return strings.Join(lines, "\n") + "\n"
}

func mergeConfig(dst *Config, src Config) {
	if strings.TrimSpace(src.ConfigPath) != "" {
		dst.ConfigPath = strings.TrimSpace(src.ConfigPath)
	}
	iface := strings.TrimSpace(src.Iface)
	if iface != "" {
		dst.Iface = iface
	}
	if value := strings.TrimSpace(src.HostIP); value != "" {
		dst.HostIP = value
	}
	if value := strings.TrimSpace(src.BoxIP); value != "" {
		dst.BoxIP = value
	}
	if value := strings.TrimSpace(src.Netmask); value != "" {
		dst.Netmask = value
	}
	if value := strings.TrimSpace(src.HardwarePortPattern); value != "" {
		dst.HardwarePortPattern = value
	}
	if value := strings.TrimSpace(src.LogLevel); value != "" {
		dst.LogLevel = value
	}
	if value := strings.TrimSpace(src.ListenAddr); value != "" {
		dst.ListenAddr = value
	}
}

func applyEnvOverrides(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("BOX_IFACE")); value != "" {
		cfg.Iface = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_HOST_IP")); value != "" {
		cfg.HostIP = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_IP")); value != "" {
		cfg.BoxIP = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_NETMASK")); value != "" {
		cfg.Netmask = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_HARDWARE_PORT_PATTERN")); value != "" {
		cfg.HardwarePortPattern = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_LOG_LEVEL")); value != "" {
		cfg.LogLevel = value
	}
	if value := strings.TrimSpace(os.Getenv("BOX_LISTEN_ADDR")); value != "" {
		cfg.ListenAddr = value
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ConfigPath) == "" {
		return fmt.Errorf("config path must not be empty")
	}
	if strings.TrimSpace(c.HostIP) == "" {
		return fmt.Errorf("host IP must not be empty")
	}
	if strings.TrimSpace(c.BoxIP) == "" {
		return fmt.Errorf("box IP must not be empty")
	}
	if strings.TrimSpace(c.Netmask) == "" {
		return fmt.Errorf("netmask must not be empty")
	}
	if strings.TrimSpace(c.HardwarePortPattern) == "" {
		return fmt.Errorf("hardware port pattern must not be empty")
	}
	if strings.TrimSpace(c.LogLevel) == "" {
		return fmt.Errorf("log level must not be empty")
	}
	if strings.TrimSpace(c.ListenAddr) == "" {
		return fmt.Errorf("listen addr must not be empty")
	}
	return nil
}
