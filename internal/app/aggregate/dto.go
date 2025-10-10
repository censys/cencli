package aggregate

import (
	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// AggregateResult represents the normalized aggregation result returned by the
// application layer. It contains response metadata and a list of buckets.
type Result struct {
	Meta    *responsemeta.ResponseMeta
	Buckets []Bucket
}

// Bucket represents a single term bucket and its count.
type Bucket struct {
	Key   string `json:"key"`
	Count uint64 `json:"count"`
}

// CountByLevel is a typed string alias describing which document level the API
// should count by when aggregating.
// Values are defined by the backend. This type is intentionally open-ended to
// avoid breaking changes if the API adds new levels.
type CountByLevel string

const (
	// CountByLevelHost counts at the host level.
	CountByLevelHost CountByLevel = "host"
	// CountByLevelService counts at the service level.
	CountByLevelService CountByLevel = "service"
	// CountByLevelProtocol counts at the protocol level.
	CountByLevelProtocol CountByLevel = "protocol"
)

// AggregateParams bundles parameters used to perform an aggregation. Using a
// struct prevents parameter drift and improves readability as the API evolves.
//
// Note: OrgID and CollectionID are options; only one of them is typically used
// at a time. If CollectionID is present, the aggregation occurs in the
// collection scope; otherwise, it occurs in the global scope.
// The CountByLevel option is a typed alias to reduce stringly-typed errors.
// The FilterByQuery option determines whether results are limited to values
// matching the query.
type Params struct {
	OrgID         mo.Option[identifiers.OrganizationID]
	CollectionID  mo.Option[identifiers.CollectionID]
	Query         string
	Field         string
	NumBuckets    int64
	CountByLevel  mo.Option[CountByLevel]
	FilterByQuery mo.Option[bool]
}

func parseBuckets(buckets []components.SearchAggregateResponseBucket) []Bucket {
	parsedBuckets := make([]Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		parsedBuckets = append(parsedBuckets, Bucket{
			Key:   bucket.Key,
			Count: uint64(bucket.Count),
		})
	}
	return parsedBuckets
}

// countByLevelToString converts a CountByLevel option to the corresponding
// string option expected by the SDK client. When not present, it returns None.
func countByLevelToString(level mo.Option[CountByLevel]) mo.Option[string] {
	if !level.IsPresent() {
		return mo.None[string]()
	}
	return mo.Some(string(level.MustGet()))
}
