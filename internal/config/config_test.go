package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCreatesDefaultConfigFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("BOX_CONFIG_PATH", "")
	t.Setenv("BOX_IFACE", "")
	t.Setenv("BOX_HOST_IP", "")
	t.Setenv("BOX_IP", "")
	t.Setenv("BOX_NETMASK", "")
	t.Setenv("BOX_HARDWARE_PORT_PATTERN", "")
	t.Setenv("BOX_LOG_LEVEL", "")
	t.Setenv("BOX_LISTEN_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantPath := filepath.Join(homeDir, DefaultConfigDirName, DefaultConfigFileName)
	if cfg.ConfigPath != wantPath {
		t.Fatalf("ConfigPath = %q, want %q", cfg.ConfigPath, wantPath)
	}
	if cfg.HostIP != DefaultHostIP || cfg.BoxIP != DefaultBoxIP || cfg.Netmask != DefaultNetmask || cfg.HardwarePortPattern != DefaultHardwarePortPattern || cfg.LogLevel != DefaultLogLevel || cfg.ListenAddr != DefaultListenAddr {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}

	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, `config_path = "`+wantPath+`"`) {
		t.Fatalf("default config does not contain config_path: %s", text)
	}
	if !strings.Contains(text, `host_ip = "`+DefaultHostIP+`"`) {
		t.Fatalf("default config does not contain host_ip: %s", text)
	}
}

func TestLoadSyncsConfigPathAndReadsValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom.toml")
	input := strings.Join([]string{
		`iface = "en7"`,
		`host_ip = "192.168.10.99"`,
		`box_ip = "192.168.10.2"`,
		`netmask = "255.255.255.0"`,
		`hardware_port_pattern = "AX88179A"`,
		`log_level = "debug"`,
		`listen_addr = "127.0.0.1:19999"`,
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(input), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("BOX_CONFIG_PATH", configPath)
	t.Setenv("BOX_IFACE", "")
	t.Setenv("BOX_HOST_IP", "")
	t.Setenv("BOX_IP", "")
	t.Setenv("BOX_NETMASK", "")
	t.Setenv("BOX_HARDWARE_PORT_PATTERN", "")
	t.Setenv("BOX_LOG_LEVEL", "")
	t.Setenv("BOX_LISTEN_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ConfigPath != configPath {
		t.Fatalf("ConfigPath = %q, want %q", cfg.ConfigPath, configPath)
	}
	if cfg.Iface != "en7" || cfg.HostIP != "192.168.10.99" || cfg.ListenAddr != "127.0.0.1:19999" || cfg.LogLevel != "DEBUG" {
		t.Fatalf("unexpected config values: %#v", cfg)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), `config_path = "`+configPath+`"`) {
		t.Fatalf("synced config missing config_path: %s", string(content))
	}
}
