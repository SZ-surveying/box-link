package netops

import (
	"bufio"
	"context"
	"regexp"
	"strings"

	"box-link/internal/config"
	"box-link/internal/iface"
	"box-link/internal/syscmd"
	"github.com/sirupsen/logrus"
)

var pingTimePattern = regexp.MustCompile(`time[=<]([0-9.]+\s*ms)`)
var pingPacketLossPattern = regexp.MustCompile(`([0-9.]+%\s+packet loss)`)

type StatusInterfaceDetails struct {
	Name    string `json:"name,omitempty"`
	IPv4    string `json:"ipv4,omitempty"`
	Netmask string `json:"netmask,omitempty"`
	Ether   string `json:"ether,omitempty"`
	Status  string `json:"status,omitempty"`
	Media   string `json:"media,omitempty"`
}

type StatusRouteDetails struct {
	Destination string `json:"destination,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Flags       string `json:"flags,omitempty"`
}

type StatusPingDetails struct {
	Target     string `json:"target,omitempty"`
	Responder  string `json:"responder,omitempty"`
	PacketLoss string `json:"packetLoss,omitempty"`
	RoundTrip  string `json:"roundTrip,omitempty"`
	Latency    string `json:"latency,omitempty"`
}

type StatusResult struct {
	Iface            string                 `json:"iface"`
	Interface        string                 `json:"interface"`
	Route            string                 `json:"route"`
	Ping             string                 `json:"ping"`
	Reachable        bool                   `json:"reachable"`
	RouteFound       bool                   `json:"routeFound"`
	InterfaceDetails StatusInterfaceDetails `json:"interfaceDetails"`
	RouteDetails     StatusRouteDetails     `json:"routeDetails"`
	PingDetails      StatusPingDetails      `json:"pingDetails"`
}

func Status(ctx context.Context, cfg config.Config, runner syscmd.Runner, logger *logrus.Logger) (StatusResult, error) {
	resolved, err := iface.Resolve(ctx, cfg, runner, logger)
	if err != nil {
		return StatusResult{}, err
	}

	logger.Debugf("mac interface: %s", resolved.Name)
	ifconfigResult, ifconfigErr := runner.Run(ctx, "ifconfig", resolved.Name)
	if ifconfigErr != nil {
		return StatusResult{}, ifconfigErr
	}

	logger.Debugf("route to box: %s", cfg.BoxIP)
	routeResult, routeErr := runner.Run(ctx, "route", "-n", "get", cfg.BoxIP)
	routeText := strings.TrimSpace(routeResult.Stdout)
	if routeErr != nil {
		routeText = strings.TrimSpace(strings.Join([]string{routeResult.Stdout, routeResult.Stderr}, "\n"))
		if routeText == "" {
			routeText = routeErr.Error()
		}
	}

	logger.Debugf("ping box: %s", cfg.BoxIP)
	pingResult, pingErr := runner.Run(ctx, "ping", "-c", "1", "-W", "1000", cfg.BoxIP)
	pingText := strings.TrimSpace(strings.Join([]string{pingResult.Stdout, pingResult.Stderr}, "\n"))
	if pingText == "" && pingErr != nil {
		pingText = pingErr.Error()
	}

	return StatusResult{
		Iface:            resolved.Name,
		Interface:        firstNLines(ifconfigResult.Stdout, 12),
		Route:            routeText,
		Ping:             pingText,
		Reachable:        pingErr == nil,
		RouteFound:       routeErr == nil,
		InterfaceDetails: parseIfconfigSummary(resolved.Name, ifconfigResult.Stdout),
		RouteDetails:     parseRouteSummary(cfg.BoxIP, routeText),
		PingDetails:      parsePingSummary(cfg.BoxIP, pingText),
	}, nil
}

func parseIfconfigSummary(name, output string) StatusInterfaceDetails {
	details := StatusInterfaceDetails{Name: name}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "inet":
			if len(fields) >= 2 && details.IPv4 == "" {
				details.IPv4 = fields[1]
			}
			for i := 2; i+1 < len(fields); i++ {
				if fields[i] == "netmask" && details.Netmask == "" {
					details.Netmask = fields[i+1]
				}
			}
		case "ether":
			if len(fields) >= 2 && details.Ether == "" {
				details.Ether = fields[1]
			}
		case "status:":
			if len(fields) >= 2 && details.Status == "" {
				details.Status = strings.Join(fields[1:], " ")
			}
		case "media:":
			if len(fields) >= 2 && details.Media == "" {
				details.Media = strings.Join(fields[1:], " ")
			}
		}
	}

	return details
}

func parseRouteSummary(target, output string) StatusRouteDetails {
	details := StatusRouteDetails{Destination: target}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "route to", "destination":
			if value != "" {
				details.Destination = value
			}
		case "gateway":
			details.Gateway = value
		case "interface":
			details.Interface = value
		case "flags":
			details.Flags = value
		}
	}

	return details
}

func parsePingSummary(target, output string) StatusPingDetails {
	details := StatusPingDetails{Target: target}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "PING ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				details.Target = fields[1]
			}
		}

		if idx := strings.Index(line, "bytes from "); idx >= 0 && details.Responder == "" {
			remainder := line[idx+len("bytes from "):]
			responder := strings.TrimSpace(strings.SplitN(remainder, ":", 2)[0])
			if responder != "" {
				details.Responder = responder
			}
		}

		if match := pingTimePattern.FindStringSubmatch(line); len(match) == 2 && details.Latency == "" {
			details.Latency = match[1]
		}

		if strings.Contains(line, "packet loss") && details.PacketLoss == "" {
			if match := pingPacketLossPattern.FindStringSubmatch(line); len(match) == 2 {
				details.PacketLoss = match[1]
			} else {
				details.PacketLoss = line
			}
		}

		if strings.HasPrefix(line, "round-trip ") || strings.HasPrefix(line, "round trip ") {
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				details.RoundTrip = strings.TrimSpace(parts[1])
			}
		}
	}

	return details
}
