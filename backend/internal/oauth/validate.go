package oauth

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxRedirects = 10

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	host := req.URL.Hostname()
	if host == "" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return validatePublicIP(ip)
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil
	}
	for _, ip := range addrs {
		if err := validatePublicIP(ip); err != nil {
			return err
		}
	}
	return nil
}

func validatePDSURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("pds url: parse: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("pds url: scheme %q must be https", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("pds url: missing host")
	}

	if ip := net.ParseIP(host); ip != nil {
		return validatePublicIP(ip)
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("pds url: resolve %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("pds url: %q resolved to no addresses", host)
	}
	for _, ip := range addrs {
		if err := validatePublicIP(ip); err != nil {
			return err
		}
	}
	return nil
}

func validatePublicIP(ip net.IP) error {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("pds url: %s is not a public address", ip)
	}

	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return fmt.Errorf("pds url: metadata endpoint blocked")
	}
	return nil
}
