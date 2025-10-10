package censeye

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/tidwall/gjson"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

const (
	defaultPrefix = "host"
)

// compileRulesForHost compiles all field-value pair rules from a host asset.
func compileRulesForHost(host *assets.Host, config *censeyeConfig) ([][]fieldValuePair, error) {
	// Marshal host resource to JSON for gjson parsing
	jstr, err := json.Marshal(host.Host)
	if err != nil {
		return nil, fmt.Errorf("error marshalling host resource: %w", err)
	}
	input := gjson.ParseBytes(jstr)

	ret := make([][]fieldValuePair, 0)

	// Compile field rules (traverses the entire JSON structure)
	if err := compileFieldRules(input, defaultPrefix, &ret, config); err != nil {
		return nil, fmt.Errorf("error compiling field rules: %w", err)
	}

	// Compile service-specific multi-field rules
	if err := compileServiceRules(input, &ret, config); err != nil {
		return nil, fmt.Errorf("error compiling service rules: %w", err)
	}

	return ret, nil
}

// compileFieldRules recursively extracts field-value pairs from JSON data.
// It handles objects, arrays, strings, and numbers.
func compileFieldRules(in gjson.Result, pfx string, ret *[][]fieldValuePair, config *censeyeConfig) error {
	switch {
	case in.IsObject():
		// Check if this object should be treated as a key-value map
		if isKeyValueObject(in, pfx, config) {
			isConfiguredPrefix := slices.Contains(config.KeyValuePrefixes, pfx)

			if isConfiguredPrefix {
				// Simple key-value pairs
				in.ForEach(func(k, v gjson.Result) bool {
					*ret = append(*ret, []fieldValuePair{
						{Field: pfx + ".key", Value: k.String()},
						{Field: pfx + ".value", Value: v.String()},
					})
					return true
				})
			} else {
				// Key-value pairs with headers array
				in.ForEach(func(k, v gjson.Result) bool {
					hdrs := v.Get("headers")
					if hdrs.Exists() && hdrs.IsArray() && len(hdrs.Array()) > 0 {
						*ret = append(*ret, []fieldValuePair{
							{Field: pfx + ".key", Value: k.String()},
							{Field: pfx + ".value", Value: hdrs.Array()[0].String()},
						})
					}
					return true
				})
			}
			return nil
		}

		// Regular object - recurse into each field
		in.ForEach(func(k, v gjson.Result) bool {
			if err := compileFieldRules(v, pfx+"."+k.String(), ret, config); err != nil {
				return false
			}
			return true
		})
	case in.IsArray():
		// Array - recurse into each element with the same prefix
		for _, subv := range in.Array() {
			if err := compileFieldRules(subv, pfx, ret, config); err != nil {
				return fmt.Errorf("error compiling field rules for array: %w", err)
			}
		}
	case in.Type == gjson.String:
		// String value - create a field-value pair
		*ret = append(*ret, []fieldValuePair{{Field: pfx, Value: in.String()}})
	case in.Type == gjson.Number:
		// Number value - create a field-value pair with raw representation
		*ret = append(*ret, []fieldValuePair{{Field: pfx, Value: in.Raw}})
	}

	return nil
}

// isKeyValueObject determines if an object should be treated as a key-value map.
func isKeyValueObject(v gjson.Result, pfx string, config *censeyeConfig) bool {
	// Check if it's in the configured key-value prefixes
	if slices.Contains(config.KeyValuePrefixes, pfx) {
		return true
	}

	if !v.IsObject() {
		return false
	}

	// Check if all values have a "headers" field (special case for HTTP headers)
	result := true
	v.ForEach(func(_, val gjson.Result) bool {
		if !val.IsObject() || !val.Get("headers").Exists() {
			result = false
			return false
		}
		return true
	})

	return result
}

// compileServiceRules generates multi-field combinations from services based on extraction rules.
func compileServiceRules(input gjson.Result, ret *[][]fieldValuePair, config *censeyeConfig) error {
	services := input.Get("services")
	if !services.Exists() || !services.IsArray() {
		return nil
	}

	for _, service := range services.Array() {
		for _, rule := range config.ExtractionRules {
			var entries []gjson.Result

			// Determine scope: either the service itself or a nested array
			if rule.Scope == "" {
				entries = []gjson.Result{service}
			} else {
				scopeVal := service.Get(rule.Scope)
				if !scopeVal.Exists() || !scopeVal.IsArray() {
					continue
				}
				entries = scopeVal.Array()
			}

			// For each entry in scope, extract all required fields
			for _, entry := range entries {
				fieldValues := make(map[string][]string)
				allExist := true

				pfx := "host.services"
				if rule.Scope != "" {
					pfx += "." + rule.Scope
				}

				// Extract values for each field in the rule
				for _, field := range rule.Fields {
					val := entry.Get(field)
					if !val.Exists() {
						allExist = false
						break
					}

					var values []string
					if val.IsArray() {
						for _, arrVal := range val.Array() {
							values = append(values, arrVal.String())
						}
					} else {
						values = append(values, val.String())
					}

					fieldValues[pfx+"."+field] = values
				}

				// Skip if not all required fields exist
				if !allExist {
					continue
				}

				// Generate all combinations of field values
				combos := genCombos(fieldValues, rule.Fields, pfx)
				*ret = append(*ret, combos...)
			}
		}
	}
	return nil
}

// genCombos generates all cartesian product combinations of field values.
func genCombos(fieldValues map[string][]string, fields []string, pfx string) [][]fieldValuePair {
	if len(fields) == 0 {
		return [][]fieldValuePair{}
	}

	fieldNames := make([]string, len(fields))
	for i, field := range fields {
		fieldNames[i] = pfx + "." + field
	}

	var combinations [][]fieldValuePair
	genCartesian(fieldValues, fieldNames, 0, []fieldValuePair{}, &combinations)

	return combinations
}

// genCartesian recursively generates the cartesian product of field values.
func genCartesian(fieldValues map[string][]string, fieldNames []string, index int, current []fieldValuePair, result *[][]fieldValuePair) {
	if index == len(fieldNames) {
		combination := make([]fieldValuePair, len(current))
		copy(combination, current)
		*result = append(*result, combination)
		return
	}

	fieldName := fieldNames[index]
	values := fieldValues[fieldName]

	for _, value := range values {
		current = append(current, fieldValuePair{
			Field: fieldName,
			Value: value,
		})
		genCartesian(fieldValues, fieldNames, index+1, current, result)
		current = current[:len(current)-1]
	}
}

// deduplicateRules removes duplicate rules based on their serialized representation.
func deduplicateRules(rules [][]fieldValuePair) [][]fieldValuePair {
	ret := make([][]fieldValuePair, 0)
	seen := make(map[string]bool)

	serialize := func(fvs []fieldValuePair) string {
		// Sort by field then value to produce stable ordering
		sortable := make([]fieldValuePair, len(fvs))
		copy(sortable, fvs)
		sort.Slice(sortable, func(i, j int) bool {
			if sortable[i].Field == sortable[j].Field {
				return sortable[i].Value < sortable[j].Value
			}
			return sortable[i].Field < sortable[j].Field
		})
		// JSON-encode the normalized slice to avoid delimiter collisions
		b, _ := json.Marshal(sortable)
		return string(b)
	}

	for _, rule := range rules {
		serialized := serialize(rule)
		if !seen[serialized] {
			seen[serialized] = true
			ret = append(ret, rule)
		}
	}

	return ret
}
