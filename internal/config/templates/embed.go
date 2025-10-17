package templates

import (
	_ "embed"
)

// these variables are only used for E2E tests
// the actual template parsing uses embed.FS

var (
	//go:embed host.hbs
	HostTemplate []byte

	//go:embed certificate.hbs
	CertificateTemplate []byte

	//go:embed webproperty.hbs
	WebPropertyTemplate []byte

	//go:embed searchresult.hbs
	SearchResultTemplate []byte
)
