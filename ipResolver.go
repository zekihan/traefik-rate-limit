package traefik_rate_limit

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type IPResolverConfig struct {
	Header   string `json:"header,omitempty"`
	UseSrcIP bool   `json:"useSrcIP,omitempty"`
}

type IPResolver struct {
	config *IPResolverConfig
	logger *PluginLogger
}

func (a *IPResolver) getIP(req *http.Request) (net.IP, error) {
	if a.config == nil {
		a.logger.Debug("No IP resolver configured")
		return a.getSrcIP(req)
	}
	if a.config.UseSrcIP || a.config.Header == "" {
		a.logger.Debug("No IP resolver configured, using source IP")
		ip, err := a.getSrcIP(req)
		if err != nil {
			return nil, fmt.Errorf("failed to parse source IP: %w", err)
		}
		return ip, nil
	}
	a.logger.Debug("Using IP resolver", slog.String("header", a.config.Header))
	ip, err := a.getIPFromHeader(req, a.config.Header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IP from header %s: %w", a.config.Header, err)
	}
	return ip, nil
}

func (a *IPResolver) getIPFromHeader(req *http.Request, header string) (net.IP, error) {
	switch header {
	case XForwardedFor:
		return a.handleXForwardedFor(req)
	default:
		return a.handleHeader(req, header)
	}
}

func (a *IPResolver) handleXForwardedFor(req *http.Request) (net.IP, error) {
	xForwardedForList := req.Header.Values(XForwardedFor)
	if len(xForwardedForList) == 1 {
		xForwardedForValuesStr := strings.Split(xForwardedForList[0], ",")
		xForwardedForValues := make([]net.IP, 0)
		if len(xForwardedForValuesStr) > 0 {
			for _, xForwardedForValue := range xForwardedForValuesStr {
				tempIP := net.ParseIP(strings.TrimSpace(xForwardedForValue))
				if tempIP != nil {
					xForwardedForValues = append(xForwardedForValues, tempIP)
				} else {
					a.logger.Debug("Invalid IP format in X-Forwarded-For", slog.String("value", xForwardedForValue))
				}
			}
		}
		for _, xForwardedForValue := range xForwardedForValues {
			if !a.isPrivateIP(xForwardedForValue) {
				a.logger.Debug("Found valid X-Forwarded-For IP", slog.String("ip", xForwardedForValue.String()))
				return xForwardedForValue, nil
			}
			a.logger.Debug("X-Forwarded-For IP is a local IP, skipping", slog.String("ip", xForwardedForValue.String()))
		}
		return nil, fmt.Errorf("no valid IP found in X-Forwarded-For")
	} else {
		return nil, fmt.Errorf("header X-Forwarded-For invalid")
	}
}

func (a *IPResolver) handleHeader(req *http.Request, header string) (net.IP, error) {
	headerValues := req.Header.Values(header)
	switch len(headerValues) {
	case 1:
		tempIP := net.ParseIP(headerValues[0])
		if tempIP == nil {
			return nil, fmt.Errorf("invalid IP format in %s: %s", header, headerValues[0])
		}
		a.logger.Debug("Found valid ip", slog.String("ip", tempIP.String()), slog.String("header", header))
		return tempIP, nil
	case 0:
		ip, err := a.getSrcIP(req)
		if err != nil {
			return nil, fmt.Errorf("failed to parse source IP: %w", err)
		}
		a.logger.Debug("No IP found in header, using source IP", slog.String("ip", ip.String()))
		return ip, nil
	default:
		return nil, fmt.Errorf("header %s invalid", header)
	}
}

func (a *IPResolver) getSrcIP(req *http.Request) (net.IP, error) {
	temp, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(temp)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP format: %s", temp)
	}
	a.logger.Debug("Parsed source IP", slog.String("ip", ip.String()))
	return ip, nil
}

func (a *IPResolver) isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	return false
}

func (a *IPResolver) getLocalIPsHardcoded() ([]*net.IPNet, error) {
	ips := make([]*net.IPNet, 0)

	localIPRanges := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local addr
		"fe80::/10",      // IPv6 link-local addr
	}
	for _, cidr := range localIPRanges {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			a.logger.Error("Error parsing CIDR", slog.String("cidr", cidr), ErrorAttrWithoutStack(err))
			return ips, err
		}
		ips = append(ips, block)
	}
	return ips, nil
}

func (a *IPResolver) isWhitelisted(ip net.IP, whitelistedIPNets []*net.IPNet) bool {
	for _, ipNet := range whitelistedIPNets {
		if ipNet.Contains(ip) {
			a.logger.Debug("IP is whitelisted", slog.String("ip", ip.String()))
			return true
		}
	}
	a.logger.Debug("IP is not whitelisted", slog.String("ip", ip.String()))
	return false
}
