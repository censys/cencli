package censyscopy

import (
	"strings"

	"github.com/censys/cencli/internal/pkg/term"
)

type CencliLink string

const (
	CencliRepo                      CencliLink = "https://github.com/censys/cencli"
	CencliDocs                      CencliLink = "https://docs.censys.com/docs/platform-cli"
	CensysPATInstructions           CencliLink = "https://docs.censys.com/reference/get-started#step-2-create-a-personal-access-token"
	CensysOrgIDInstructions         CencliLink = "https://docs.censys.com/reference/get-started#step-3-find-and-use-your-organization-id-optional"
	CensysHostLookupTemplate        CencliLink = "https://platform.censys.io/hosts/{host_id}"
	CensysCertificateLookupTemplate CencliLink = "https://platform.censys.io/certificates/{certificate_id}"
	CensysWebPropertyLookupTemplate CencliLink = "https://platform.censys.io/webproperties/{hostname:port}"
)

func (l CencliLink) String() string {
	return string(l)
}

// Render returns a terminal-friendly clickable link.
// This should only be used if the result is being written to a TTY.
// If no anchor text is provided, the link's URL will be used as
// the anchor text, with the protocol stripped.
func (l CencliLink) Render(anchorText string) string {
	url := string(l)
	if anchorText == "" {
		anchorText = strings.TrimPrefix(url, "https://")
	}
	return term.RenderLink(url, anchorText)
}

func CensysHostLookupLink(hostID string) CencliLink {
	return CencliLink(strings.Replace(string(CensysHostLookupTemplate), "{host_id}", hostID, 1))
}

func CensysCertificateLookupLink(certID string) CencliLink {
	return CencliLink(strings.Replace(string(CensysCertificateLookupTemplate), "{certificate_id}", certID, 1))
}

func CensysWebPropertyLookupLink(hostport string) CencliLink {
	return CencliLink(strings.Replace(string(CensysWebPropertyLookupTemplate), "{hostname:port}", hostport, 1))
}
