// Package jq provides a minimal jq-style path evaluator using only the standard library.
//
// Supported syntax:
//
//	.             identity — returns the input value unchanged
//	.field        access an object key
//	.field.sub    chain key accesses
//	.field[]      access key, then iterate over the array value
//	.[]           iterate over a top-level array
//	.a[].b        combinations of the above
package jq

import (
	"encoding/json"
	"fmt"
	"strings"
)

type step struct {
	field  string // empty = operate on current value (used for .[])
	expand bool   // true = iterate over array after field access
}

// Parse parses a jq-style path expression into a slice of steps.
// Returns nil steps for "." (identity).
func Parse(expr string) ([]step, error) {
	if expr == "" || expr == "." {
		return nil, nil
	}
	if !strings.HasPrefix(expr, ".") {
		return nil, fmt.Errorf("jq expression must start with '.' (got %q)", expr)
	}
	expr = expr[1:] // strip leading dot

	var steps []step
	for _, part := range strings.Split(expr, ".") {
		if part == "" {
			continue
		}
		switch {
		case part == "[]":
			steps = append(steps, step{expand: true})
		case strings.HasSuffix(part, "[]"):
			steps = append(steps, step{field: part[:len(part)-2], expand: true})
		default:
			steps = append(steps, step{field: part})
		}
	}
	return steps, nil
}

// eval applies parsed steps to an already-decoded JSON value (any = map[string]any | []any | primitives).
func eval(steps []step, v any) []any {
	if len(steps) == 0 {
		return []any{v}
	}
	s, rest := steps[0], steps[1:]

	current := v
	if s.field != "" {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = m[s.field]
	}

	var items []any
	if s.expand {
		arr, ok := current.([]any)
		if !ok {
			return nil
		}
		items = arr
	} else {
		items = []any{current}
	}

	var results []any
	for _, item := range items {
		results = append(results, eval(rest, item)...)
	}
	return results
}

// Eval evaluates expr against v (a JSON-decoded Go value) and returns the matching values.
func Eval(expr string, v any) ([]any, error) {
	steps, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	return eval(steps, v), nil
}

// EvalJSON evaluates expr against raw JSON and returns the matching values.
func EvalJSON(expr string, data []byte) ([]any, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return Eval(expr, v)
}

// FormatValue formats a jq result value for output, mirroring jq -r behaviour:
// strings are printed without quotes; all other types use compact JSON.
func FormatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
