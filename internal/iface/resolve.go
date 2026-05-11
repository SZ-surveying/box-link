package iface

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"box-link/internal/config"
	"box-link/internal/syscmd"
	"github.com/sirupsen/logrus"
)

var enPattern = regexp.MustCompile(`^en[0-9]+$`)

type Result struct {
	Name   string `json:"name"`
	Method string `json:"method"`
}

func Resolve(ctx context.Context, cfg config.Config, runner syscmd.Runner, logger *logrus.Logger) (Result, error) {
	logger.Debugf("resolving box interface")

	if cfg.Iface != "" {
		if interfaceExists(ctx, runner, cfg.Iface) {
			logger.Debugf("matched via BOX_IFACE=%s", cfg.Iface)
			return Result{Name: cfg.Iface, Method: "BOX_IFACE"}, nil
		}
		logger.Errorf("BOX_IFACE is set but does not exist: %s", cfg.Iface)
		return Result{}, fmt.Errorf("BOX_IFACE is set but does not exist: %s", cfg.Iface)
	}

	logger.Debugf("trying hardware port match with pattern: %s", cfg.HardwarePortPattern)
	if name := matchHardwarePort(ctx, cfg, runner); name != "" {
		if interfaceExists(ctx, runner, name) {
			logger.Debugf("matched via hardware port pattern %q: %s", cfg.HardwarePortPattern, name)
			return Result{Name: name, Method: "hardware-port"}, nil
		}
	}
	logger.Debugf("no usable interface matched hardware port pattern %q", cfg.HardwarePortPattern)

	logger.Debugf("trying interface with static IP %s", cfg.HostIP)
	if name := matchInterfaceByIP(ctx, cfg.HostIP, runner); name != "" {
		logger.Debugf("matched via static IP %s: %s", cfg.HostIP, name)
		return Result{Name: name, Method: "static-ip"}, nil
	}

	logger.Debugf("trying active external en* interface")
	if name := matchActiveExternal(ctx, runner); name != "" {
		logger.Debugf("matched via active external interface: %s", name)
		return Result{Name: name, Method: "active-external"}, nil
	}

	logger.Debugf("trying highest-numbered external en* interface")
	if name := matchHighestExternal(ctx, runner); name != "" {
		logger.Debugf("matched via highest-numbered external interface fallback: %s", name)
		return Result{Name: name, Method: "highest-external"}, nil
	}

	logger.Errorf("could not find a usable external en* interface")
	return Result{}, fmt.Errorf("could not find a usable external en* interface")
}

func interfaceExists(ctx context.Context, runner syscmd.Runner, iface string) bool {
	_, err := runner.Run(ctx, "ifconfig", iface)
	return err == nil
}

func matchHardwarePort(ctx context.Context, cfg config.Config, runner syscmd.Runner) string {
	result, err := runner.Run(ctx, "networksetup", "-listallhardwareports")
	if err != nil {
		return ""
	}

	var matched bool
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "Hardware Port: "):
			port := strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port: "))
			matched = strings.Contains(port, cfg.HardwarePortPattern)
		case matched && strings.HasPrefix(line, "Device: "):
			return strings.TrimSpace(strings.TrimPrefix(line, "Device: "))
		}
	}
	return ""
}

func matchInterfaceByIP(ctx context.Context, hostIP string, runner syscmd.Runner) string {
	result, err := runner.Run(ctx, "ifconfig")
	if err != nil {
		return ""
	}

	blocks := splitIfconfigBlocks(result.Stdout)
	for _, block := range blocks {
		if !enPattern.MatchString(block.name) {
			continue
		}
		for _, line := range block.lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "inet" && fields[1] == hostIP {
				return block.name
			}
		}
	}
	return ""
}

func matchActiveExternal(ctx context.Context, runner syscmd.Runner) string {
	result, err := runner.Run(ctx, "ifconfig")
	if err != nil {
		return ""
	}

	blocks := splitIfconfigBlocks(result.Stdout)
	bestName := ""
	bestRank := -1
	for _, block := range blocks {
		if block.name == "en0" || !enPattern.MatchString(block.name) {
			continue
		}
		if !hasStatusActive(block.lines) {
			continue
		}
		rank := interfaceRank(block.name)
		if rank > bestRank {
			bestName = block.name
			bestRank = rank
		}
	}
	return bestName
}

func matchHighestExternal(ctx context.Context, runner syscmd.Runner) string {
	result, err := runner.Run(ctx, "ifconfig", "-l")
	if err != nil {
		return ""
	}

	var names []string
	for _, field := range strings.Fields(result.Stdout) {
		if field == "en0" || !enPattern.MatchString(field) {
			continue
		}
		names = append(names, field)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Slice(names, func(i, j int) bool {
		return interfaceRank(names[i]) > interfaceRank(names[j])
	})
	return names[0]
}

type ifconfigBlock struct {
	name  string
	lines []string
}

func splitIfconfigBlocks(input string) []ifconfigBlock {
	var blocks []ifconfigBlock
	var current *ifconfigBlock

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, "\t") && strings.Contains(trimmed, ":") {
			name := strings.TrimSuffix(strings.SplitN(trimmed, ":", 2)[0], ":")
			block := ifconfigBlock{name: name, lines: []string{trimmed}}
			blocks = append(blocks, block)
			current = &blocks[len(blocks)-1]
			continue
		}
		if current != nil {
			current.lines = append(current.lines, trimmed)
		}
	}

	return blocks
}

func hasStatusActive(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == "status: active" {
			return true
		}
	}
	return false
}

func interfaceRank(name string) int {
	n := strings.TrimPrefix(name, "en")
	rank, err := strconv.Atoi(n)
	if err != nil {
		return -1
	}
	return rank
}
