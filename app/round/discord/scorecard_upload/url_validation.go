package scorecardupload

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func validateUDiscURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Errorf("url is empty")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("url must use https")
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "" {
		return fmt.Errorf("url host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		return fmt.Errorf("ip hosts are not allowed")
	}
	if host != "udisc.com" && !strings.HasSuffix(host, ".udisc.com") {
		return fmt.Errorf("url host must be on udisc.com")
	}

	return nil
}
