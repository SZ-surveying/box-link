package netops

import (
	"context"
	"fmt"
	"os"

	"box-link/internal/config"
	"box-link/internal/iface"
	"box-link/internal/syscmd"
	"github.com/sirupsen/logrus"
)

type OffResult struct {
	Iface    string `json:"iface"`
	Ifconfig string `json:"ifconfig"`
}

func Off(ctx context.Context, cfg config.Config, runner syscmd.Runner, logger *logrus.Logger) (OffResult, error) {
	if os.Geteuid() != 0 {
		return OffResult{}, fmt.Errorf("box-link off requires administrator privileges; run with sudo")
	}

	resolved, err := iface.Resolve(ctx, cfg, runner, logger)
	if err != nil {
		return OffResult{}, err
	}

	logger.Infof("restoring %s to DHCP", resolved.Name)
	if _, err := runner.Run(ctx, "ipconfig", "set", resolved.Name, "DHCP"); err != nil {
		return OffResult{}, err
	}

	show, err := runner.Run(ctx, "ifconfig", resolved.Name)
	if err != nil {
		return OffResult{}, err
	}

	return OffResult{
		Iface:    resolved.Name,
		Ifconfig: firstNLines(show.Stdout, 8),
	}, nil
}
