package responsemeta

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResponseMeta is a sanitized, application-level representation of an HTTP interaction.
// It avoids exposing raw http.Request/Response to higher layers and strips credentials.
type ResponseMeta struct {
	Method     string
	URL        string
	Status     int
	Latency    time.Duration
	Headers    map[string]string
	PageCount  uint64
	RetryCount uint64
}

// NewResponseMeta constructs a ResponseMeta for printing or logging purposes.
func NewResponseMeta(request *http.Request, response *http.Response, latency time.Duration, attempts uint64) *ResponseMeta {
	meta := &ResponseMeta{
		Latency:    latency,
		Headers:    make(map[string]string),
		RetryCount: 0,
	}

	if attempts > 1 {
		meta.RetryCount = attempts - 1
	}

	if request != nil {
		meta.Method = request.Method
		meta.URL = sanitizedURL(request.URL)
		for k, v := range sanitizedHeaders(request.Header, nil) {
			meta.Headers[k] = v
		}
	}

	if response != nil {
		meta.Status = response.StatusCode
		for k, v := range sanitizedHeaders(nil, response.Header) {
			meta.Headers[k] = v
		}
	}

	return meta
}

func sanitizedURL(url *url.URL) string {
	if url == nil {
		return ""
	}
	clone := *url
	clone.User = nil
	return clone.String()
}

func sanitizedHeaders(reqHeaders http.Header, resHeaders http.Header) map[string]string {
	joined := make(map[string]string)
	if len(reqHeaders) > 0 {
		for k, v := range reqHeaders {
			joined["req-"+k] = strings.Join(v, ", ")
		}
	}
	if len(resHeaders) > 0 {
		for k, v := range resHeaders {
			joined["res-"+k] = strings.Join(v, ", ")
		}
	}
	sensitiveHeaders := map[string]struct{}{
		"authorization":       {},
		"cookie":              {},
		"set-cookie":          {},
		"x-api-key":           {},
		"x-auth-token":        {},
		"proxy-authorization": {},
	}
	sanitized := make(map[string]string)
	for k, v := range joined {
		parts := strings.SplitN(k, "-", 2)
		headerKey := strings.ToLower(parts[len(parts)-1])
		if _, ok := sensitiveHeaders[headerKey]; ok {
			sanitized[k] = "********"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}
