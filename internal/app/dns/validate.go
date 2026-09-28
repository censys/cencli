package dns

import (
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/operations"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// nameRecordTypes is the set of accepted --record-type values for a name
// lookup, sourced from the SDK's generated enum so it stays in sync with the
// API contract.
var nameRecordTypes = []string{
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesA),
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesAaaa),
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesMx),
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesNs),
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesSoa),
	string(operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypesTxt),
}

// ipRecordTypes is the set of accepted --record-type values for an IP lookup,
// sourced from the SDK's generated enum.
var ipRecordTypes = []string{
	string(operations.RecordTypesA),
	string(operations.RecordTypesAaaa),
}

// normalizeRecordTypes uppercases, trims, and de-duplicates the given record
// types, and rejects any that the lookup direction does not support. It
// returns nil for no types, which the API reads as all supported types.
func normalizeRecordTypes(values []string, supported []string) ([]string, cenclierrors.CencliError) {
	var out []string
	for _, v := range values {
		upper := strings.ToUpper(strings.TrimSpace(v))
		if upper == "" {
			continue
		}
		if !slices.Contains(supported, upper) {
			return nil, NewInvalidRecordTypeError(strings.TrimSpace(v), supported)
		}
		if !slices.Contains(out, upper) {
			out = append(out, upper)
		}
	}
	return out, nil
}

// validatePaginationParams rejects pagination values that would fetch nothing.
func validatePaginationParams(pageSize, maxPages mo.Option[uint64]) cenclierrors.CencliError {
	if pageSize.IsPresent() && pageSize.MustGet() == 0 {
		return NewInvalidPaginationParamsError("page size must be greater than 0")
	}
	if maxPages.IsPresent() && maxPages.MustGet() == 0 {
		return NewInvalidPaginationParamsError("max pages must be greater than 0")
	}
	return nil
}
