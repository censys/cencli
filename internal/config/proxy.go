package config

// ProxyConfig holds proxy and transport settings for outbound HTTP requests.
type ProxyConfig struct {
	// URL overrides HTTP_PROXY/HTTPS_PROXY env vars. Supported schemes: http, https, socks5, socks5h.
	URL string `yaml:"url" mapstructure:"url" doc:"Proxy URL for outbound requests (e.g. http://proxy.example.com:8080, socks5://proxy.example.com:1080). Overrides HTTP_PROXY/HTTPS_PROXY environment variables. Supported schemes: http, https, socks5, socks5h."`
	// DisableHTTP2 forces HTTP/1.1. Useful when the proxy or network does not support HTTP/2.
	DisableHTTP2 bool `yaml:"disable-http2" mapstructure:"disable-http2" doc:"Disable HTTP/2. Use if your proxy or network infrastructure doesn't support HTTP/2."`
}

var defaultProxyConfig = ProxyConfig{}
