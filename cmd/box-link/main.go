package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"box-link/internal/app"
	"box-link/internal/config"
	"box-link/internal/web"
)

var version = "0.1.0"

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		printUsage(os.Stdout)
		return 0
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(version)
		return 0
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	application := app.New(cfg)

	switch os.Args[1] {
	case "iface":
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		result, err := application.ResolveIface(ctx)
		if err != nil {
			return 1
		}
		fmt.Println(result.Name)
		return 0
	case "on":
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		result, err := application.On(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(result.Ifconfig)
		return 0
	case "off":
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		result, err := application.Off(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(result.Ifconfig)
		return 0
	case "status":
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		result, err := application.Status(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("=== mac interface ===")
		fmt.Println(result.Interface)
		fmt.Println()
		fmt.Println("=== route to box ===")
		fmt.Println(result.Route)
		fmt.Println()
		fmt.Println("=== ping box ===")
		fmt.Println(result.Ping)
		return 0
	case "doctor":
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		result, err := application.Doctor(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, check := range result.Checks {
			fmt.Printf("[%s] %s: %s\n", check.Level, check.Name, check.Detail)
		}
		return 0
	case "info":
		fmt.Printf("Config Path: %s\n", cfg.ConfigPath)
		fmt.Printf("Iface override: %s\n", displayOrAuto(cfg.Iface))
		fmt.Printf("Host IP: %s\n", cfg.HostIP)
		fmt.Printf("Box IP: %s\n", cfg.BoxIP)
		fmt.Printf("Netmask: %s\n", cfg.Netmask)
		fmt.Printf("Hardware match: %s\n", cfg.HardwarePortPattern)
		fmt.Printf("Log level: %s\n", cfg.LogLevel)
		fmt.Printf("Listen: %s\n", cfg.ListenAddr)
		return 0
	case "ui":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		server := web.New(application, cfg.ListenAddr)
		if err := server.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `box-link

Direct-link box networking helper for macOS.

Usage:
  box-link <command>
  box-link --help
  box-link --version

Commands:
  info      Show the active config file path and resolved settings
  iface     Resolve which interface box-link will use
  status    Show interface, route, and ping status
  doctor    Run diagnostics and print a concise summary
  on        Configure the interface with the box-link static IP
  off       Restore the interface back to DHCP
  ui        Start the local web console
  version   Print the version

Config:
  Default config file: ~/.box-link/config.toml
  The file is created automatically on first run with starter values.
  Environment variables such as BOX_IFACE still override file values.

Common examples:
  box-link info
  box-link iface
  box-link status
  sudo box-link on
  sudo box-link off
  sudo box-link ui

Web UI:
  Default listen address: 127.0.0.1:18888
  Open http://127.0.0.1:18888 after starting 'box-link ui'.`)
}

func displayOrAuto(value string) string {
	if value == "" {
		return "(auto)"
	}
	return value
}
