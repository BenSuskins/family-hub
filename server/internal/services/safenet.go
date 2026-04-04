package services

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ValidateExternalURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("disallowed scheme: %s", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}

	if isBlockedHost(hostname) {
		return fmt.Errorf("disallowed host: %s", hostname)
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("resolving host: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("disallowed IP address: %s resolves to %s", hostname, ip)
		}
	}

	return nil
}

func isBlockedHost(hostname string) bool {
	lower := strings.ToLower(hostname)
	blocked := []string{
		"localhost",
		"metadata.google.internal",
		"169.254.169.254",
	}
	for _, blocked := range blocked {
		if lower == blocked {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{parseCIDR("10.0.0.0/8")},
		{parseCIDR("172.16.0.0/12")},
		{parseCIDR("192.168.0.0/16")},
		{parseCIDR("127.0.0.0/8")},
		{parseCIDR("169.254.0.0/16")},
		{parseCIDR("::1/128")},
		{parseCIDR("fc00::/7")},
		{parseCIDR("fe80::/10")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(cidr string) *net.IPNet {
	_, network, _ := net.ParseCIDR(cidr)
	return network
}
