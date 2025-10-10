package censys

import (
	"net/http"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
)

type Metadata struct {
	Request  *http.Request
	Response *http.Response
	Latency  time.Duration
	Attempts uint64
}

type Result[T any] struct {
	Metadata Metadata
	Data     *T
}

type responseEnvelope interface {
	GetHTTPMeta() components.HTTPMetadata
}

func buildResponseMetadata(responseEnvelope responseEnvelope, latency time.Duration, attempts uint64) Metadata {
	httpMeta := responseEnvelope.GetHTTPMeta()
	return Metadata{
		Request:  httpMeta.Request,
		Response: httpMeta.Response,
		Latency:  latency,
		Attempts: attempts,
	}
}
