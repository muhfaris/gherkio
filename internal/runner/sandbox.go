package runner

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

// ValidateURL checks whether a URL conforms to the configured sandboxing security rules.
func ValidateURL(rawURL string, cfg *model.SandboxConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL syntax: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		// Fallback for relative URLs or URLs parsed without host
		host = parsed.Path
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
	}

	// 1. Resolve host IP addresses to prevent DNS rebinding bypasses
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			// Check private subnet blocking
			if cfg.BlockPrivateSubnets && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
				return fmt.Errorf("private or local loopback IP %s is blocked", ip.String())
			}
		}
	}

	// 2. Extra secure fallback for direct localhost/IP matching
	if cfg.BlockPrivateSubnets {
		directIP := net.ParseIP(host)
		if directIP != nil {
			if directIP.IsLoopback() || directIP.IsPrivate() || directIP.IsLinkLocalUnicast() {
				return fmt.Errorf("private or local loopback IP %s is blocked", directIP.String())
			}
		} else {
			hostLower := strings.ToLower(host)
			if hostLower == "localhost" || hostLower == "127.0.0.1" || hostLower == "::1" {
				return fmt.Errorf("private or local loopback IP is blocked")
			}
		}
	}

	// 3. Match against explicitly blocked domains list
	for _, blockedPattern := range cfg.BlockedDomains {
		if matchDomain(host, blockedPattern) {
			return fmt.Errorf("domain %s is explicitly blocked", host)
		}
	}

	// 4. Match against allowed domains list (if list is not empty)
	if len(cfg.AllowedDomains) > 0 {
		allowed := false
		for _, allowedPattern := range cfg.AllowedDomains {
			if matchDomain(host, allowedPattern) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain %s is not in the allowed domains list", host)
		}
	}

	return nil
}

// matchDomain checks if the host matches the pattern (supporting simple wildcards).
func matchDomain(host, pattern string) bool {
	// Strip port suffix from both host and pattern
	host = stripPort(host)
	pattern = stripPort(pattern)

	hostLower := strings.ToLower(host)
	patternLower := strings.ToLower(pattern)

	if patternLower == "*" {
		return true
	}

	// Exact match
	if hostLower == patternLower {
		return true
	}

	// Wildcard matching (e.g. *.example.com)
	if strings.HasPrefix(patternLower, "*.") {
		suffix := patternLower[1:] // ".example.com"
		if len(hostLower) >= len(suffix) && strings.HasSuffix(hostLower, suffix) {
			return true
		}
	}

	return false
}

// stripPort removes the port component from host or pattern if it exists.
func stripPort(hostOrPattern string) string {
	if idx := strings.Index(hostOrPattern, ":"); idx != -1 {
		return hostOrPattern[:idx]
	}
	return hostOrPattern
}
