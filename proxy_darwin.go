package sysproxy

/*
#cgo CFLAGS: -mmacosx-version-min=10.10
#cgo LDFLAGS: -framework CoreFoundation -framework SystemConfiguration

#include <CoreFoundation/CoreFoundation.h>
#include <SystemConfiguration/SystemConfiguration.h>
#include <stdlib.h>

typedef enum { PROXY_HTTP, PROXY_HTTPS, PROXY_SOCKS } ProxyType;

typedef struct {
    int enabled;
    char host[256];
    int port;
} ProxyInfo;

ProxyInfo getProxy(CFDictionaryRef settings, ProxyType type) {
    ProxyInfo info = {0};
    if (!settings) return info;

    CFStringRef enableKey, hostKey;
    CFStringRef portKey;

    switch (type) {
    case PROXY_HTTP:
        enableKey = kSCPropNetProxiesHTTPEnable;
        hostKey   = kSCPropNetProxiesHTTPProxy;
        portKey   = kSCPropNetProxiesHTTPPort;
        break;
    case PROXY_HTTPS:
        enableKey = kSCPropNetProxiesHTTPSEnable;
        hostKey   = kSCPropNetProxiesHTTPSProxy;
        portKey   = kSCPropNetProxiesHTTPSPort;
        break;
    case PROXY_SOCKS:
        enableKey = kSCPropNetProxiesSOCKSEnable;
        hostKey   = kSCPropNetProxiesSOCKSProxy;
        portKey   = kSCPropNetProxiesSOCKSPort;
        break;
    default:
        return info;
    }

    CFNumberRef enabledVal = CFDictionaryGetValue(settings, enableKey);
    if (enabledVal && CFGetTypeID(enabledVal) == CFNumberGetTypeID()) {
        CFNumberGetValue(enabledVal, kCFNumberIntType, &info.enabled);
    }

    if (info.enabled) {
        CFStringRef hostVal = CFDictionaryGetValue(settings, hostKey);
        CFNumberRef portVal = CFDictionaryGetValue(settings, portKey);

        if (hostVal && CFGetTypeID(hostVal) == CFStringGetTypeID()) {
            CFStringGetCString(hostVal, info.host, sizeof(info.host), kCFStringEncodingUTF8);
        }
        if (portVal && CFGetTypeID(portVal) == CFNumberGetTypeID()) {
            CFNumberGetValue(portVal, kCFNumberIntType, &info.port);
        }
    }

    return info;
}
*/
import "C"
import (
	"fmt"
)

type setting struct {
	ref C.CFDictionaryRef
}

func (s *setting) Close() {
	var null C.CFDictionaryRef
	if s.ref != null {
		C.CFRelease(C.CFTypeRef(s.ref))
	}
}

func createSetting() (*setting, error) {
	var null C.SCDynamicStoreRef
	setRef := C.SCDynamicStoreCopyProxies(null)
	var nullDict C.CFDictionaryRef
	if setRef == nullDict {
		return nil, fmt.Errorf("failed to get system proxy settings")
	}

	return &setting{
		ref: C.CFDictionaryRef(setRef),
	}, nil
}

func getProxy(settings C.CFDictionaryRef, proxyType C.ProxyType, scheme string) *Item {
	raw := C.getProxy(settings, proxyType)
	if raw.enabled == 0 {
		return nil
	}

	return &Item{
		Scheme: scheme,
		Host:   C.GoString(&raw.host[0]),
		Port:   uint16(raw.port),
	}
}

func GetHTTP() (*Item, error) {
	s, err := createSetting()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	return getProxy(s.ref, C.PROXY_HTTP, "http"), nil
}

func GetHTTPS() (*Item, error) {
	s, err := createSetting()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	return getProxy(s.ref, C.PROXY_HTTPS, "https"), nil
}

func GetSOCKS() (*Item, error) {
	s, err := createSetting()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	return getProxy(s.ref, C.PROXY_SOCKS, "socks5"), nil
}

func GetAll() (httpProxy, httpsProxy, socksProxy *Item, err error) {
	s, err := createSetting()
	if err != nil {
		return nil, nil, nil, err
	}
	defer s.Close()

	httpProxy = getProxy(s.ref, C.PROXY_HTTP, "http")
	httpsProxy = getProxy(s.ref, C.PROXY_HTTPS, "https")
	socksProxy = getProxy(s.ref, C.PROXY_SOCKS, "socks5")
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
