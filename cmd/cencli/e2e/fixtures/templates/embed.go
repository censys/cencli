package templates

import (
	_ "embed"
)

// files in this directory will be copied by the Makefile before running E2E tests
// they remain checked in to the repository to ensure CI doesn't report this
// file as having errors if the templates are missing.

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
