package sysproxy

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetProxy  *syscall.LazyProc
	procGlobalFree *syscall.LazyProc
)

func init() {
	// Ref: https://learn.microsoft.com/en-us/windows/win32/api/winhttp/nf-winhttp-winhttpgetieproxyconfigforcurrentuser
	procGetProxy = syscall.NewLazyDLL("winhttp.dll").NewProc("WinHttpGetIEProxyConfigForCurrentUser")
	procGlobalFree = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalFree")
}

// Ref: https://learn.microsoft.com/en-us/windows/win32/api/winhttp/ns-winhttp-winhttp_current_user_ie_proxy_config
type rawProxyConfig struct {
	autoDetect    int32 // BOOL is 4 bytes on Windows, not 1
	autoConfigUrl *uint16
	proxy         *uint16
	proxyBypass   *uint16
}

func getRawProxyConfig() (*rawProxyConfig, error) {
	var c rawProxyConfig
	r1, _, err := procGetProxy.Call(uintptr(unsafe.Pointer(&c)))
	if r1 == 0 {
		return nil, fmt.Errorf("cannot get IE proxy config: %w", err)
	}
	return &c, nil
}

func globalFree(p *uint16) {
	if p != nil {
		procGlobalFree.Call(uintptr(unsafe.Pointer(p)))
	}
}

func (c *rawProxyConfig) free() {
	globalFree(c.autoConfigUrl)
	globalFree(c.proxy)
	globalFree(c.proxyBypass)
}

// parseHostPort parses "host:port" into an *Item.
func parseHostPort(s string) *Item {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Strip any scheme prefix (e.g. "http://", "socks=")
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}

	part := strings.SplitN(s, ":", 2)
	if len(part) != 2 {
		return nil
	}

	host := part[0]
	port, err := strconv.ParseUint(part[1], 10, 16)
	if err != nil {
		return nil
	}

	return &Item{
		Host: host,
		Port: uint16(port),
	}
}

// parseProxyString parses Windows proxy string which can be either:
//   - "host:port" (applies to all protocols)
//   - "http=host:port;https=host:port;socks=host:port" (protocol-specific)
func parseProxyString(raw string) (httpProxy, httpsProxy, socksProxy *Item) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil, nil
	}

	// Check if it contains "=" which indicates protocol-specific format
	if !strings.Contains(raw, "=") {
		// Simple format: host:port applies to HTTP and HTTPS
		item := parseHostPort(raw)
		return item, item, nil
	}

	// Protocol-specific format: "http=host:port;https=host:port;socks=host:port"
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}

		proto := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		item := parseHostPort(value)

		switch proto {
		case "http":
			httpProxy = item
		case "https":
			httpsProxy = item
		case "socks":
			socksProxy = item
		}
	}

	return
}

func GetHTTP() (*Item, error) {
	c, err := getRawProxyConfig()
	if err != nil {
		return nil, err
	}
	defer c.free()

	proxyURL := windows.UTF16PtrToString(c.proxy)
	http, _, _ := parseProxyString(proxyURL)
	return http, nil
}

func GetHTTPS() (*Item, error) {
	c, err := getRawProxyConfig()
	if err != nil {
		return nil, err
	}
	defer c.free()

	proxyURL := windows.UTF16PtrToString(c.proxy)
	_, https, _ := parseProxyString(proxyURL)
	return https, nil
}

func GetSOCKS() (*Item, error) {
	c, err := getRawProxyConfig()
	if err != nil {
		return nil, err
	}
	defer c.free()

	proxyURL := windows.UTF16PtrToString(c.proxy)
	_, _, socks := parseProxyString(proxyURL)
	return socks, nil
}

func GetAll() (httpProxy, httpsProxy, socksProxy *Item, err error) {
	c, err := getRawProxyConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	defer c.free()

	proxyURL := windows.UTF16PtrToString(c.proxy)
	httpProxy, httpsProxy, socksProxy = parseProxyString(proxyURL)
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
