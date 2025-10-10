package assets

import (
	"github.com/censys/censys-sdk-go/models/components"
)

type Asset interface {
	AssetType() AssetType
}

// Certificate represents a certificate asset.
// This has 1:1 correspondence with the SDK's Certificate type.
type Certificate struct{ components.Certificate }

func (c Certificate) AssetType() AssetType { return AssetTypeCertificate }

var _ Asset = Certificate{}

func NewCertificate(cert components.Certificate) Certificate { return Certificate{cert} }

// Host represents a host asset.
// This has 1:1 correspondence with the SDK's Host type.
type Host struct {
	components.Host
	MatchedServices []components.MatchedService `json:"matched_services,omitempty"`
}

func (h Host) AssetType() AssetType { return AssetTypeHost }

var _ Asset = Host{}

func NewHostWithMatchedServices(host components.Host, matchedServices []components.MatchedService) Host {
	return Host{host, matchedServices}
}

func NewHost(host components.Host) Host { return Host{host, nil} }

// WebProperty represents a web property asset.
// This has 1:1 correspondence with the SDK's Webproperty type.
type WebProperty struct{ components.Webproperty }

func (w WebProperty) AssetType() AssetType { return AssetTypeWebProperty }

var _ Asset = WebProperty{}

func NewWebProperty(webProperty components.Webproperty) WebProperty { return WebProperty{webProperty} }
