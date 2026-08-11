package provider

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"testing"
)

type staticResolver map[string][]net.IP

func (r staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	values := r[host]
	result := make([]net.IPAddr, 0, len(values))
	for _, value := range values {
		result = append(result, net.IPAddr{IP: value})
	}
	return result, nil
}

func TestValidateBaseURL(t *testing.T) {
	t.Parallel()
	resolver := staticResolver{
		"provider.example": {net.ParseIP("8.8.8.8")},
		"mixed.example": {
			net.ParseIP("8.8.8.8"),
			net.ParseIP("10.0.0.1"),
		},
	}

	t.Run("accepts public HTTPS target", func(t *testing.T) {
		t.Parallel()
		target, err := ValidateBaseURL(context.Background(), "https://provider.example/openai", OutboundPolicy{
			Resolver: resolver,
		})
		if err != nil {
			t.Fatalf("ValidateBaseURL() error = %v", err)
		}
		if target.String() != "https://provider.example/openai" {
			t.Fatalf("ValidateBaseURL() = %q", target.String())
		}
	})

	tests := []struct {
		name   string
		target string
	}{
		{name: "plain HTTP", target: "http://provider.example"},
		{name: "loopback literal", target: "https://127.0.0.1"},
		{name: "metadata address", target: "https://169.254.169.254"},
		{name: "private DNS result", target: "https://mixed.example"},
		{name: "local hostname", target: "https://service.local"},
		{name: "credentials", target: "https://user:secret@provider.example"},
		{name: "base query", target: "https://provider.example?token=secret"},
	}
	for _, test := range tests {
		test := test
		t.Run("rejects "+test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateBaseURL(context.Background(), test.target, OutboundPolicy{
				Resolver: resolver,
			})
			if err == nil {
				t.Fatalf("ValidateBaseURL(%q) unexpectedly succeeded", test.target)
			}
		})
	}
}

func TestParseResourceURLAllowsSignedQuery(t *testing.T) {
	t.Parallel()
	target, err := parseResourceURL(
		"https://cdn.example/image.png?signature=secret",
		OutboundPolicy{},
	)
	if err != nil {
		t.Fatalf("parseResourceURL() error = %v", err)
	}
	if target.RawQuery != "signature=secret" {
		t.Fatalf("RawQuery = %q", target.RawQuery)
	}
}

func TestSameOriginIncludesSchemeAndPort(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://provider.example/v1")
	same, _ := url.Parse("https://provider.example/v1/models")
	otherPort, _ := url.Parse("https://provider.example:8443/v1/models")
	plainHTTP, _ := url.Parse("http://provider.example/v1/models")
	if !sameOrigin(base, same) {
		t.Fatal("same origin was rejected")
	}
	if sameOrigin(base, otherPort) || sameOrigin(base, plainHTTP) {
		t.Fatal("scheme or port change was treated as the same origin")
	}
}

func TestValidateIPRejectsReservedRanges(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"0.0.0.1",
		"10.0.0.1",
		"100.64.0.1",
		"127.0.0.1",
		"169.254.169.254",
		"192.0.2.1",
		"198.18.0.1",
		"203.0.113.1",
		"::1",
		"fc00::1",
		"fe80::1",
		"2001:db8::1",
	} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			ip := net.ParseIP(value)
			if ip == nil {
				t.Fatalf("invalid test IP %q", value)
			}
			if err := validateResolvedIP(ip); err == nil {
				t.Fatalf("validateResolvedIP(%q) unexpectedly succeeded", value)
			}
		})
	}
}

func validateResolvedIP(value net.IP) error {
	ip, ok := netip.AddrFromSlice(value)
	if !ok {
		return nil
	}
	return validateIP(ip.Unmap(), false)
}
