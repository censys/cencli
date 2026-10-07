package dns

import (
	"slices"
	"strings"

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

// ValidateRecordTypes checks --record-type values against the set the lookup
// direction supports (isIP selects the IP set over the name set). It is
// exported so the command can reject an unsupported record type in PreRun,
// before a service (and so the API client) is needed; the service runs the
// same check again for callers that invoke it directly.
func ValidateRecordTypes(recordTypes []string, isIP bool) cenclierrors.CencliError {
	supported := nameRecordTypes
	if isIP {
		supported = ipRecordTypes
	}
	_, err := normalizeRecordTypes(recordTypes, supported)
	return err
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
