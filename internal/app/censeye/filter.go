package censeye

import (
	"strings"
)

// filterSpecKind represents the type of filter matching to apply.
type filterSpecKind int

const (
	filterSpecExact filterSpecKind = iota
	filterSpecPrefix
)

// filterSpec is a typed representation of a filter rule.
type filterSpec struct {
	kind  filterSpecKind
	field string
}

// parseFilterSpecs converts human-friendly string filter rules into typed specs.
// A trailing dot indicates a prefix filter; otherwise it is an exact match filter.
func parseFilterSpecs(filters []string) []filterSpec {
	if len(filters) == 0 {
		return nil
	}
	specs := make([]filterSpec, 0, len(filters))
	for _, f := range filters {
		if strings.HasSuffix(f, ".") {
			specs = append(specs, filterSpec{kind: filterSpecPrefix, field: f})
			continue
		}
		specs = append(specs, filterSpec{kind: filterSpecExact, field: f})
	}
	return specs
}

// applyFilters applies all configured filters and returns the optimized rule set.
func applyFilters(rules [][]fieldValuePair, config *censeyeConfig) [][]fieldValuePair {
	filtered := applyPrefixFilters(rules, config.Filters)
	filtered = applyRegexFilters(filtered, config)
	filtered = deduplicateRules(filtered)
	return filtered
}

// applyPrefixFilters removes rules that match any of the configured filter prefixes.
func applyPrefixFilters(rules [][]fieldValuePair, filters []string) [][]fieldValuePair {
	filtered := make([][]fieldValuePair, 0)
	specs := parseFilterSpecs(filters)

	for _, rule := range rules {
		shouldSkip := false

		for _, fv := range rule {
		specsLoop:
			for _, spec := range specs {
				switch spec.kind {
				case filterSpecPrefix:
					if strings.HasPrefix(fv.Field, spec.field) {
						shouldSkip = true
						break specsLoop
					}
				case filterSpecExact:
					if fv.Field == spec.field {
						shouldSkip = true
						break specsLoop
					}
				}
			}
			if shouldSkip {
				break
			}
		}

		if !shouldSkip {
			filtered = append(filtered, rule)
		}
	}

	return filtered
}

// applyRegexFilters removes rules whose CenQL query matches any of the regex filters.
func applyRegexFilters(pairs [][]fieldValuePair, config *censeyeConfig) [][]fieldValuePair {
	ret := make([][]fieldValuePair, 0)

	for _, pair := range pairs {
		cql := toCenqlQuery(pair)

		skip := false
		for _, rgx := range config.RgxFilters {
			if rgx.MatchString(cql) {
				skip = true
				break
			}
		}

		if !skip {
			ret = append(ret, pair)
		}
	}

	return ret
}
