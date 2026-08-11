package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type DialContextFunc func(context.Context, string, string) (net.Conn, error)

type OutboundPolicy struct {
	AllowHTTP           bool
	AllowPrivateNetwork bool
	Resolver            Resolver
	DialContext         DialContextFunc
	ConnectTimeout      time.Duration
}

func ValidateBaseURL(ctx context.Context, rawURL string, policy OutboundPolicy) (*url.URL, error) {
	target, err := parseOutboundURL(rawURL, policy)
	if err != nil {
		return nil, err
	}
	if err := validateResolvedTarget(ctx, target, policy); err != nil {
		return nil, err
	}
	return target, nil
}

func parseOutboundURL(rawURL string, policy OutboundPolicy) (*url.URL, error) {
	return parseURL(rawURL, policy, false)
}

func parseResourceURL(rawURL string, policy OutboundPolicy) (*url.URL, error) {
	return parseURL(rawURL, policy, true)
}

func parseURL(rawURL string, policy OutboundPolicy, allowQuery bool) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse provider base URL: %w", err)
	}
	if !target.IsAbs() || target.Hostname() == "" {
		return nil, errors.New("provider base URL must be absolute")
	}
	switch strings.ToLower(target.Scheme) {
	case "https":
	case "http":
		if !policy.AllowHTTP {
			return nil, errors.New("provider base URL must use HTTPS")
		}
	default:
		return nil, errors.New("provider base URL uses an unsupported scheme")
	}
	if target.User != nil {
		return nil, errors.New("provider base URL must not contain user information")
	}
	if (!allowQuery && target.RawQuery != "") || target.Fragment != "" {
		return nil, errors.New("provider base URL must not contain a query or fragment")
	}
	if strings.Contains(target.EscapedPath(), "..") {
		return nil, errors.New("provider base URL path must not contain parent traversal")
	}
	if err := validateHostname(target.Hostname(), policy.AllowPrivateNetwork); err != nil {
		return nil, err
	}
	target.Path = strings.TrimRight(target.Path, "/")
	return target, nil
}

func validateResolvedTarget(ctx context.Context, target *url.URL, policy OutboundPolicy) error {
	host := target.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		return validateIP(ip.Unmap(), policy.AllowPrivateNetwork)
	}

	resolver := policy.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return errors.New("provider host could not be resolved")
	}
	if len(addresses) == 0 {
		return errors.New("provider host did not resolve to an address")
	}
	for _, address := range addresses {
		ip, ok := netip.AddrFromSlice(address.IP)
		if !ok {
			return errors.New("provider host resolved to an invalid address")
		}
		if err := validateIP(ip.Unmap(), policy.AllowPrivateNetwork); err != nil {
			return err
		}
	}
	return nil
}

func validateHostname(host string, allowPrivate bool) error {
	normalized := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if normalized == "" {
		return errors.New("provider host is required")
	}
	if allowPrivate {
		return nil
	}
	if normalized == "localhost" ||
		strings.HasSuffix(normalized, ".localhost") ||
		strings.HasSuffix(normalized, ".local") ||
		strings.HasSuffix(normalized, ".internal") {
		return errors.New("provider host points to a private network")
	}
	return nil
}

func validateIP(ip netip.Addr, allowPrivate bool) error {
	if !ip.IsValid() {
		return errors.New("provider host resolved to an invalid address")
	}
	if allowPrivate {
		return nil
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return errors.New("provider host points to a non-public address")
	}

	// Reject special ranges not covered by the netip helpers, including CGNAT,
	// benchmarking, documentation-only and IPv4 broadcast destinations.
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(ip) {
			return errors.New("provider host points to a reserved address")
		}
	}
	return nil
}

var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"2001:db8::/32",
	"2001:2::/48",
)

func mustPrefixes(values ...string) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		result = append(result, netip.MustParsePrefix(value))
	}
	return result
}

func newSafeHTTPClient(policy OutboundPolicy, responseHeaderTimeout, requestTimeout time.Duration, maxRedirects int) *http.Client {
	connectTimeout := policy.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 5 * time.Second
	}
	baseDialer := &net.Dialer{
		Timeout:   connectTimeout,
		KeepAlive: 30 * time.Second,
	}
	dial := policy.DialContext
	if dial == nil {
		dial = baseDialer.DialContext
	}
	if policy.Resolver == nil {
		policy.Resolver = net.DefaultResolver
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safeDialContext(policy, dial),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   connectTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	if maxRedirects <= 0 {
		maxRedirects = 3
	}
	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("provider redirect limit exceeded")
			}
			if _, err := parseResourceURL(req.URL.String(), policy); err != nil {
				return errors.New("provider redirected to an unsafe URL")
			}
			if len(via) > 0 && !sameOrigin(req.URL, via[len(via)-1].URL) {
				req.Header.Del("Authorization")
			}
			return nil
		},
	}
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Hostname(), right.Hostname()) &&
		portForURL(left) == portForURL(right)
}

func safeDialContext(policy OutboundPolicy, dial DialContextFunc) DialContextFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("invalid outbound address")
		}
		target := &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}
		if err := validateHostname(host, policy.AllowPrivateNetwork); err != nil {
			return nil, err
		}

		addresses := []net.IPAddr{}
		if parsed, parseErr := netip.ParseAddr(host); parseErr == nil {
			addresses = append(addresses, net.IPAddr{IP: net.IP(parsed.AsSlice())})
		} else {
			resolved, resolveErr := policy.Resolver.LookupIPAddr(ctx, target.Hostname())
			if resolveErr != nil {
				return nil, errors.New("outbound host could not be resolved")
			}
			addresses = resolved
		}

		var lastErr error
		for _, address := range addresses {
			ip, ok := netip.AddrFromSlice(address.IP)
			if !ok {
				continue
			}
			ip = ip.Unmap()
			if err := validateIP(ip, policy.AllowPrivateNetwork); err != nil {
				return nil, err
			}
			conn, dialErr := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr != nil {
			return nil, errors.New("provider connection failed")
		}
		return nil, errors.New("outbound host did not resolve to a usable address")
	}
}

func portForURL(target *url.URL) string {
	if port := target.Port(); port != "" {
		return port
	}
	if strings.EqualFold(target.Scheme, "http") {
		return strconv.Itoa(80)
	}
	return strconv.Itoa(443)
}
