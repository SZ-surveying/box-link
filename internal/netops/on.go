package netops

import (
	"context"
	"fmt"
	"os"
	"strings"

	"box-link/internal/config"
	"box-link/internal/iface"
	"box-link/internal/syscmd"
	"github.com/sirupsen/logrus"
)

type OnResult struct {
	Iface     string `json:"iface"`
	HostIP    string `json:"hostIP"`
	Netmask   string `json:"netmask"`
	BoxIP     string `json:"boxIP"`
	Ifconfig  string `json:"ifconfig"`
	Reachable bool   `json:"reachable"`
}

func On(ctx context.Context, cfg config.Config, runner syscmd.Runner, logger *logrus.Logger) (OnResult, error) {
	if os.Geteuid() != 0 {
		return OnResult{}, fmt.Errorf("box-link on requires administrator privileges; run with sudo")
	}

	resolved, err := iface.Resolve(ctx, cfg, runner, logger)
	if err != nil {
		return OnResult{}, err
	}

	logger.Infof("configuring %s -> %s/%s", resolved.Name, cfg.HostIP, cfg.Netmask)
	if _, err := runner.Run(ctx, "ifconfig", resolved.Name, "inet", cfg.HostIP, "netmask", cfg.Netmask, "up"); err != nil {
		return OnResult{}, err
	}

	show, err := runner.Run(ctx, "ifconfig", resolved.Name)
	if err != nil {
		return OnResult{}, err
	}

	logger.Infof("testing box reachability: %s", cfg.BoxIP)
	_, pingErr := runner.Run(ctx, "ping", "-c", "1", "-W", "1000", cfg.BoxIP)
	reachable := pingErr == nil
	if reachable {
		logger.Infof("box reachable at %s", cfg.BoxIP)
	} else {
		logger.Warnf("box not reachable yet at %s", cfg.BoxIP)
	}

	return OnResult{
		Iface:     resolved.Name,
		HostIP:    cfg.HostIP,
		Netmask:   cfg.Netmask,
		BoxIP:     cfg.BoxIP,
		Ifconfig:  firstNLines(show.Stdout, 8),
		Reachable: reachable,
	}, nil
}

func firstNLines(input string, n int) string {
	lines := strings.Split(strings.TrimRight(input, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:n], "\n")
}
