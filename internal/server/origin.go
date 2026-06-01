package server

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ParseTrustedOrigins(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		origin, err := normalizeTrustedOrigin(part)
		if err != nil {
			return nil, err
		}
		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins, nil
}

func TrustedOriginsForAddrs(addrs ...string) []string {
	seen := map[string]bool{}
	origins := []string{}

	addOrigin := func(host, port string) {
		if host == "" || port == "" {
			return
		}

		origin := "http://" + strings.ToLower(net.JoinHostPort(host, port))
		if seen[origin] {
			return
		}

		seen[origin] = true
		origins = append(origins, origin)
	}

	addLoopbackOrigins := func(port string) {
		addOrigin("localhost", port)
		addOrigin("127.0.0.1", port)
		addOrigin("::1", port)
	}

	for _, addr := range addrs {
		host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
		if err != nil {
			continue
		}

		if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
			addLoopbackOrigins(port)
			continue
		}

		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
		if strings.EqualFold(host, "localhost") {
			addLoopbackOrigins(port)
			continue
		}

		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			addLoopbackOrigins(port)
			continue
		}

		addOrigin(host, port)
	}

	return origins
}

func normalizeTrustedOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid trusted origin %q: %w", raw, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("trusted origin %q must use http or https", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("trusted origin %q must include a host", raw)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("trusted origin %q must not include path, query, userinfo, or fragment", raw)
	}

	return scheme + "://" + strings.ToLower(parsed.Host), nil
}
