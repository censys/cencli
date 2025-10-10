package tree

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// s/o @CrosleyZack

const (
	maxSummaryLength     = 80
	defaultExpandedDepth = 0 // Depth level to expand by default (0 = only root level)
)

// node represents a node in the JSON tree
type node struct {
	Key      string
	Value    string
	Children []*node
	Parent   *node
	Expanded bool
	IsLeaf   bool
}

// escapeString properly escapes a string for display, converting newlines and other special characters
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// generateObjectSummary recursively extracts all leaf values from an object and concatenates them
func generateObjectSummary(obj map[string]any) string {
	var values []string
	extractLeafValues(obj, &values)

	summary := strings.Join(values, " ")
	if len(summary) > maxSummaryLength {
		if maxSummaryLength > 3 {
			summary = summary[:maxSummaryLength-3] + "..."
		} else {
			summary = "..."
		}
	}

	return summary
}

// generateArraySummary creates a comma-separated summary for arrays containing only leaf nodes
func generateArraySummary(arr []any) string {
	var values []string
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			values = append(values, v)
		case float64:
			values = append(values, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			values = append(values, strconv.FormatBool(v))
		case nil:
			values = append(values, "null")
		default:
			values = append(values, fmt.Sprintf("%v", v))
		}
	}

	summary := strings.Join(values, ", ")
	if len(summary) > maxSummaryLength {
		if maxSummaryLength > 3 {
			summary = summary[:maxSummaryLength-3] + "..."
		} else {
			summary = "..."
		}
	}

	return summary
}

// isArrayOfLeafNodes checks if an array contains only leaf nodes (no nested objects or arrays)
func isArrayOfLeafNodes(arr []any) bool {
	for _, item := range arr {
		switch item.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

// extractLeafValues recursively extracts all leaf values from a data structure
func extractLeafValues(data any, values *[]string) {
	switch v := data.(type) {
	case map[string]any:
		for _, value := range v {
			extractLeafValues(value, values)
		}
	case []any:
		for _, item := range v {
			extractLeafValues(item, values)
		}
	case string:
		if v != "" {
			*values = append(*values, escapeString(v))
		}
	case float64:
		*values = append(*values, strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		*values = append(*values, strconv.FormatBool(v))
	case nil:
		// Skip null values
	default:
		str := fmt.Sprintf("%v", v)
		if str != "" {
			*values = append(*values, str)
		}
	}
}

// parseNodes converts any data into a tree of nodes
func parseNodes(data any) []*node {
	switch v := data.(type) {
	case map[string]any:
		return parseObject(v, nil, 0)
	case []any:
		root := &node{
			Key:      "data",
			Expanded: true, // Root wrapper is always expanded to show array contents
			IsLeaf:   false,
		}
		if isArrayOfLeafNodes(v) {
			root.Value = generateArraySummary(v)
		} else {
			root.Value = fmt.Sprintf("array[%d]", len(v))
		}
		root.Children = parseArray(v, root, 1) // Start array children at depth 1
		return []*node{root}
	default:
		return []*node{{
			Key:    "data",
			Value:  fmt.Sprintf("%v", v),
			IsLeaf: true,
		}}
	}
}

// parseObject converts a JSON object to nodes
func parseObject(obj map[string]any, parent *node, depth int) []*node {
	nodes := make([]*node, 0, len(obj))

	// Sort keys alphabetically for deterministic output
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := obj[key]
		node := &node{
			Key:      key,
			Parent:   parent,
			Expanded: depth <= defaultExpandedDepth,
		}

		switch v := value.(type) {
		case map[string]any:
			node.Value = generateObjectSummary(v)
			node.IsLeaf = false
			node.Children = parseObject(v, node, depth+1)
		case []any:
			if isArrayOfLeafNodes(v) {
				node.Value = generateArraySummary(v)
			} else {
				node.Value = fmt.Sprintf("array[%d]", len(v))
			}
			node.IsLeaf = false
			node.Children = parseArray(v, node, depth+1)
		case string:
			node.Value = fmt.Sprintf("\"%s\"", escapeString(v))
			node.IsLeaf = true
		case float64:
			node.Value = strconv.FormatFloat(v, 'f', -1, 64)
			node.IsLeaf = true
		case bool:
			node.Value = strconv.FormatBool(v)
			node.IsLeaf = true
		case nil:
			node.Value = "null"
			node.IsLeaf = true
		default:
			node.Value = fmt.Sprintf("%v", v)
			node.IsLeaf = true
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// parseArray converts a JSON array to nodes
func parseArray(arr []any, parent *node, depth int) []*node {
	nodes := make([]*node, 0, len(arr))

	for i, value := range arr {
		node := &node{
			Key:      strconv.Itoa(i),
			Parent:   parent,
			Expanded: depth <= defaultExpandedDepth,
		}

		switch v := value.(type) {
		case map[string]any:
			node.Value = generateObjectSummary(v)
			node.IsLeaf = false
			node.Children = parseObject(v, node, depth+1)
		case []any:
			if isArrayOfLeafNodes(v) {
				node.Value = generateArraySummary(v)
			} else {
				node.Value = fmt.Sprintf("array[%d]", len(v))
			}
			node.IsLeaf = false
			node.Children = parseArray(v, node, depth+1)
		case string:
			node.Value = fmt.Sprintf("\"%s\"", escapeString(v))
			node.IsLeaf = true
		case float64:
			node.Value = strconv.FormatFloat(v, 'f', -1, 64)
			node.IsLeaf = true
		case bool:
			node.Value = strconv.FormatBool(v)
			node.IsLeaf = true
		case nil:
			node.Value = "null"
			node.IsLeaf = true
		default:
			node.Value = fmt.Sprintf("%v", v)
			node.IsLeaf = true
		}

		nodes = append(nodes, node)
	}

	return nodes
}
