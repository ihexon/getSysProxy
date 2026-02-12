package sysproxy

import (
	"net/url"
	"os"
	"strconv"
)

func getEnv(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

func parseProxyURL(raw string) *Item {
	if raw == "" {
		return nil
	}

	// Try parsing as a URL first
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		host := u.Hostname()
		portStr := u.Port()
		if portStr == "" {
			return nil
		}
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return nil
		}
		return &Item{Host: host, Port: uint16(port)}
	}

	// Fallback: try bare "host:port"
	if host, portStr, ok := splitHostPort(raw); ok {
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return nil
		}
		return &Item{Host: host, Port: uint16(port)}
	}

	return nil
}

func splitHostPort(s string) (host, port string, ok bool) {
	last := len(s) - 1
	if last < 0 {
		return "", "", false
	}

	// Find last colon
	i := last
	for i >= 0 && s[i] != ':' {
		i--
	}
	if i < 0 {
		return "", "", false
	}

	return s[:i], s[i+1:], true
}

func GetHTTP() (*Item, error) {
	raw := getEnv("http_proxy", "HTTP_PROXY")
	return parseProxyURL(raw), nil
}

func GetHTTPS() (*Item, error) {
	raw := getEnv("https_proxy", "HTTPS_PROXY")
	return parseProxyURL(raw), nil
}

func GetSOCKS() (*Item, error) {
	raw := getEnv("all_proxy", "ALL_PROXY")
	return parseProxyURL(raw), nil
}

func GetAll() (httpProxy, httpsProxy, socksProxy *Item, err error) {
	httpProxy = parseProxyURL(getEnv("http_proxy", "HTTP_PROXY"))
	httpsProxy = parseProxyURL(getEnv("https_proxy", "HTTPS_PROXY"))
	socksProxy = parseProxyURL(getEnv("all_proxy", "ALL_PROXY"))
	return
}

// IsEnabled returns true if any system proxy (HTTP, HTTPS, or SOCKS) is configured.
func IsEnabled() (bool, error) {
	h, hs, s, err := GetAll()
	if err != nil {
		return false, err
	}
	return h != nil || hs != nil || s != nil, nil
}
