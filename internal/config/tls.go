package config

// TLSConfig holds TLS and certificate settings for outbound HTTP requests.
type TLSConfig struct {
	// CABundle is appended to the system CA pool for TLS verification.
	CABundle string `yaml:"ca-bundle" mapstructure:"ca-bundle" doc:"Path to a PEM-encoded CA bundle file. Appended to the system CA pool for TLS verification. Supports ~ and environment variables."`
	// InsecureSkipVerify disables TLS certificate verification entirely.
	InsecureSkipVerify bool `yaml:"insecure-skip-verify" mapstructure:"insecure-skip-verify" doc:"Disable TLS certificate verification. Insecure — only use when you cannot import the CA."`
	// ClientCert is the path to a PEM-encoded client certificate for mTLS.
	ClientCert string `yaml:"client-cert" mapstructure:"client-cert" doc:"Path to a PEM-encoded client certificate for mTLS authentication. Supports ~ and environment variables."`
	// ClientKey is the path to the private key paired with ClientCert.
	ClientKey string `yaml:"client-key" mapstructure:"client-key" doc:"Path to the PEM-encoded private key paired with client-cert. Supports ~ and environment variables."`
}

var defaultTLSConfig = TLSConfig{}
