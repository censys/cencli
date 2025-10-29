package helpers

import "github.com/censys/cencli/internal/pkg/censyscopy"

type lookupOperation string

const (
	lookupOperationHost        lookupOperation = "host"
	lookupOperationCertificate lookupOperation = "certificate"
	lookupOperationWebProperty lookupOperation = "webproperty"
)

func (o lookupOperation) String() string {
	return string(o)
}

func parseLookupOperation(op string) lookupOperation {
	switch op {
	case lookupOperationHost.String():
		return lookupOperationHost
	case lookupOperationCertificate.String():
		return lookupOperationCertificate
	case lookupOperationWebProperty.String():
		return lookupOperationWebProperty
	default:
		return ""
	}
}

type lookupURLHelper struct {
	render bool
}

var _ HandlebarsHelper = &lookupURLHelper{}

// NewLookupURLHelper creates a helper that returns a link to the lookup URL for the given operation and ID.
// Callers must set render to false if the result is not being written to a TTY.
func NewLookupURLHelper(render bool) HandlebarsHelper {
	return &lookupURLHelper{render: render}
}

func (h *lookupURLHelper) Name() string {
	return "platform_lookup_url"
}

func (h *lookupURLHelper) Function() any {
	return func(op, id string) string {
		var link censyscopy.CencliLink
		switch parseLookupOperation(op) {
		case lookupOperationHost:
			link = censyscopy.CensysHostLookupLink(id)
		case lookupOperationCertificate:
			link = censyscopy.CensysCertificateLookupLink(id)
		case lookupOperationWebProperty:
			link = censyscopy.CensysWebPropertyLookupLink(id)
		default:
			return ""
		}
		if h.render {
			return link.Render("")
		}
		return link.String()
	}
}
