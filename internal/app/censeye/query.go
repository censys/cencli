package censeye

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	servicesPrefix = "host.services"
	platformURL    = "https://platform.censys.io/search?q="
)

// toCenqlQuery converts field-value pairs into a CenQL query string.
func toCenqlQuery(pairs []fieldValuePair) string {
	if len(pairs) == 0 {
		return ""
	}

	// Single field-value pair
	if len(pairs) == 1 {
		return fmt.Sprintf("%s=%q", pairs[0].Field, pairs[0].Value)
	}

	// Multiple field-value pairs
	out := make([]string, len(pairs))
	for i, pair := range pairs {
		field := strings.TrimPrefix(pair.Field, servicesPrefix+".")
		out[i] = fmt.Sprintf("%s=%q", field, pair.Value)
	}

	return fmt.Sprintf("%s:(%s)", servicesPrefix, strings.Join(out, " and "))
}

// toSearchURL converts a CenQL query into a Censys platform search URL.
func toSearchURL(query string) string {
	return platformURL + url.QueryEscape(query)
}
