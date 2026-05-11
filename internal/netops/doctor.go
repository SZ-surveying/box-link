package netops

import (
	"context"
	"fmt"
	"strings"

	"box-link/internal/config"
	"box-link/internal/iface"
	"box-link/internal/syscmd"
	"github.com/sirupsen/logrus"
)

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Level  string `json:"level"`
}

type DoctorResult struct {
	Iface  string  `json:"iface"`
	Checks []Check `json:"checks"`
}

func Doctor(ctx context.Context, cfg config.Config, runner syscmd.Runner, logger *logrus.Logger) (DoctorResult, error) {
	result := DoctorResult{Checks: make([]Check, 0, 5)}

	resolved, err := iface.Resolve(ctx, cfg, runner, logger)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "interface resolution",
			OK:     false,
			Level:  "ERROR",
			Detail: err.Error(),
		})
		return result, nil
	}
	result.Iface = resolved.Name
	result.Checks = append(result.Checks, Check{
		Name:   "interface resolution",
		OK:     true,
		Level:  "INFO",
		Detail: fmt.Sprintf("resolved %s via %s", resolved.Name, resolved.Method),
	})

	ifconfigResult, err := runner.Run(ctx, "ifconfig", resolved.Name)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "interface query",
			OK:     false,
			Level:  "ERROR",
			Detail: err.Error(),
		})
	} else if strings.Contains(ifconfigResult.Stdout, "inet "+cfg.HostIP+" ") {
		result.Checks = append(result.Checks, Check{
			Name:   "host IP",
			OK:     true,
			Level:  "INFO",
			Detail: fmt.Sprintf("%s is configured on %s", cfg.HostIP, resolved.Name),
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "host IP",
			OK:     false,
			Level:  "WARN",
			Detail: fmt.Sprintf("%s is not currently configured on %s", cfg.HostIP, resolved.Name),
		})
	}

	routeResult, routeErr := runner.Run(ctx, "route", "-n", "get", cfg.BoxIP)
	if routeErr != nil {
		detail := strings.TrimSpace(strings.Join([]string{routeResult.Stdout, routeResult.Stderr}, "\n"))
		if detail == "" {
			detail = routeErr.Error()
		}
		result.Checks = append(result.Checks, Check{
			Name:   "route lookup",
			OK:     false,
			Level:  "WARN",
			Detail: detail,
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "route lookup",
			OK:     true,
			Level:  "INFO",
			Detail: firstNLines(routeResult.Stdout, 3),
		})
	}

	_, pingErr := runner.Run(ctx, "ping", "-c", "1", "-W", "1000", cfg.BoxIP)
	if pingErr != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "box reachability",
			OK:     false,
			Level:  "WARN",
			Detail: fmt.Sprintf("box is not reachable at %s", cfg.BoxIP),
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "box reachability",
			OK:     true,
			Level:  "INFO",
			Detail: fmt.Sprintf("box is reachable at %s", cfg.BoxIP),
		})
	}

	return result, nil
}
