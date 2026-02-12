package sysproxy

import "fmt"

type Item struct {
	Scheme string
	Host   string
	Port   uint16
}

// String returns the proxy as a URL string, e.g. "http://127.0.0.1:8080".
func (i *Item) String() string {
	return fmt.Sprintf("%s://%s:%d", i.Scheme, i.Host, i.Port)
}
