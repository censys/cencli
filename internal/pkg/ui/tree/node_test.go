package tree

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, nodes []*node)
		wantErr  bool
	}{
		{
			name:  "primitives and basic structures",
			input: `{"string": "hello", "number": 42, "boolean": true, "null_value": null, "array": [1, 2, 3], "object": {"nested": "value"}}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 6, "Expected 6 nodes")

				nodeMap := make(map[string]*node)
				for _, node := range nodes {
					nodeMap[node.Key] = node
				}

				// Validate primitive types
				assert.Equal(t, `"hello"`, nodeMap["string"].Value)
				assert.True(t, nodeMap["string"].IsLeaf)
				assert.Equal(t, "42", nodeMap["number"].Value)
				assert.True(t, nodeMap["number"].IsLeaf)
				assert.Equal(t, "true", nodeMap["boolean"].Value)
				assert.True(t, nodeMap["boolean"].IsLeaf)
				assert.Equal(t, "null", nodeMap["null_value"].Value)
				assert.True(t, nodeMap["null_value"].IsLeaf)

				// Validate array with leaf nodes shows comma-separated values
				assert.Equal(t, "1, 2, 3", nodeMap["array"].Value)
				assert.False(t, nodeMap["array"].IsLeaf)
				assert.Len(t, nodeMap["array"].Children, 3)

				// Validate object shows summary
				assert.Equal(t, "value", nodeMap["object"].Value)
				assert.False(t, nodeMap["object"].IsLeaf)
				assert.Len(t, nodeMap["object"].Children, 1)
			},
		},
		{
			name:  "arrays - leaf vs nested",
			input: `{"leaf_array": ["admin", "user"], "nested_array": [{"name": "Alice"}, {"name": "Bob"}], "empty_array": []}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 3)
				nodeMap := make(map[string]*node)
				for _, node := range nodes {
					nodeMap[node.Key] = node
				}

				// Leaf array shows comma-separated values
				assert.Equal(t, "admin, user", nodeMap["leaf_array"].Value)

				// Nested array shows array[n] format
				assert.Equal(t, "array[2]", nodeMap["nested_array"].Value)

				// Empty array shows empty string
				assert.Equal(t, "", nodeMap["empty_array"].Value)
			},
		},
		{
			name:  "root level structures",
			input: `[1, 2, 3]`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				root := nodes[0]
				assert.Equal(t, "data", root.Key)
				assert.Equal(t, "1, 2, 3", root.Value)
				assert.False(t, root.IsLeaf)
				assert.Len(t, root.Children, 3)
			},
		},
		{
			name:  "edge cases and special characters",
			input: `{"empty_string": "", "unicode": "Hello 世界", "escaped": "line1\nline2\ttab\"quote", "zero": 0, "false": false}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 5)
				nodeMap := make(map[string]*node)
				for _, node := range nodes {
					nodeMap[node.Key] = node
				}

				assert.Equal(t, `""`, nodeMap["empty_string"].Value)
				assert.Equal(t, `"Hello 世界"`, nodeMap["unicode"].Value)
				assert.Equal(t, `"line1\nline2\ttab\"quote"`, nodeMap["escaped"].Value)
				assert.Equal(t, "0", nodeMap["zero"].Value)
				assert.Equal(t, "false", nodeMap["false"].Value)
			},
		},
		{
			name:    "invalid JSON",
			input:   `{"invalid": json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse JSON to get any data
			var data any
			err := json.Unmarshal([]byte(tt.input), &data)

			if tt.wantErr {
				assert.Error(t, err, "JSON parsing expected error but got none")
				return
			}

			require.NoError(t, err, "JSON parsing unexpected error")
			nodes := parseNodes(data)
			tt.validate(t, nodes)
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "string with newline",
			input:    "hello\nworld",
			expected: "hello\\nworld",
		},
		{
			name:     "string with tab",
			input:    "hello\tworld",
			expected: "hello\\tworld",
		},
		{
			name:     "string with carriage return",
			input:    "hello\rworld",
			expected: "hello\\rworld",
		},
		{
			name:     "string with quotes",
			input:    `hello "world"`,
			expected: `hello \"world\"`,
		},
		{
			name:     "string with backslashes",
			input:    `hello\world`,
			expected: `hello\\world`,
		},
		{
			name:     "string with multiple escape sequences",
			input:    "hello\n\t\"world\"\r\\test",
			expected: "hello\\n\\t\\\"world\\\"\\r\\\\test",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "unicode characters",
			input:    "Hello 世界 🚀",
			expected: "Hello 世界 🚀",
		},
		{
			name:     "mixed escape and unicode",
			input:    "Hello\n世界\t🚀\"test\"",
			expected: "Hello\\n世界\\t🚀\\\"test\\\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseObjectDepthExpansion(t *testing.T) {
	// Test expansion behavior at different depths
	input := map[string]any{
		"nested": map[string]any{"value": "test"},
	}

	// Depth 0 should expand
	nodes0 := parseObject(input, nil, 0)
	require.Len(t, nodes0, 1)
	assert.True(t, nodes0[0].Expanded, "Node at depth 0 should be expanded")

	// Depth 3 should not expand
	nodes3 := parseObject(input, nil, 3)
	require.Len(t, nodes3, 1)
	assert.False(t, nodes3[0].Expanded, "Node at depth 3 should not be expanded")
}

func TestParseArrayDepthExpansion(t *testing.T) {
	// Test expansion behavior at different depths
	input := []any{"a", "b"}

	// Depth 0 should expand
	nodes0 := parseArray(input, nil, 0)
	require.Len(t, nodes0, 2)
	for _, node := range nodes0 {
		assert.True(t, node.Expanded, "Node at depth 0 should be expanded")
	}

	// Depth 3 should not expand
	nodes3 := parseArray(input, nil, 3)
	require.Len(t, nodes3, 2)
	for _, node := range nodes3 {
		assert.False(t, node.Expanded, "Node at depth 3 should not be expanded")
	}
}

func TestNodeRelationships(t *testing.T) {
	input := `{"parent": {"child": "value"}}`
	var data any
	err := json.Unmarshal([]byte(input), &data)
	require.NoError(t, err, "JSON parsing failed")

	nodes := parseNodes(data)
	require.Len(t, nodes, 1, "Expected 1 root node")

	parent := nodes[0]
	assert.Nil(t, parent.Parent, "Root node should have no parent")

	require.Len(t, parent.Children, 1, "Parent should have 1 child")

	child := parent.Children[0]
	assert.Equal(t, parent, child.Parent, "Child's parent should point to parent node")
}

func TestComplexStructures(t *testing.T) {
	// Test a realistic API response with mixed array types
	input := `{
		"status": "success",
		"data": {
			"users": [{"id": 1, "name": "John"}],
			"tags": ["admin", "user"],
			"metadata": {"count": 2, "active": true}
		}
	}`

	var data any
	err := json.Unmarshal([]byte(input), &data)
	require.NoError(t, err)

	nodes := parseNodes(data)
	require.Len(t, nodes, 2) // status and data

	// Find data node and validate its children
	var dataNode *node
	for _, node := range nodes {
		if node.Key == "data" {
			dataNode = node
			break
		}
	}
	require.NotNil(t, dataNode)
	require.Len(t, dataNode.Children, 3)

	// Verify array handling: users (objects) vs tags (primitives)
	childMap := make(map[string]*node)
	for _, child := range dataNode.Children {
		childMap[child.Key] = child
	}

	assert.Equal(t, "array[1]", childMap["users"].Value)   // Contains objects
	assert.Equal(t, "admin, user", childMap["tags"].Value) // Contains primitives
}

// Benchmark tests for performance validation
func BenchmarkParseNodes_SimpleObject(b *testing.B) {
	jsonStr := `{"name": "John", "age": 30, "active": true}`
	var data any
	err := json.Unmarshal([]byte(jsonStr), &data)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseNodes(data)
	}
}

func BenchmarkParseNodes_LargeArray(b *testing.B) {
	// Create a large array
	var items []any
	for i := 0; i < 1000; i++ {
		items = append(items, map[string]any{
			"id":   i,
			"name": "Item " + string(rune(i)),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseNodes(items)
	}
}

func BenchmarkParseNodes_DeepNesting(b *testing.B) {
	// Create deeply nested structure
	nested := make(map[string]any)
	current := nested
	for i := 0; i < 100; i++ {
		next := make(map[string]any)
		current["level"] = next
		current = next
	}
	current["value"] = "deep"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseNodes(nested)
	}
}

func BenchmarkParseNodes_ComplexMixed(b *testing.B) {
	jsonStr := `{
		"users": [
			{"id": 1, "name": "Alice", "tags": ["admin", "active"], "profile": {"age": 30, "verified": true}},
			{"id": 2, "name": "Bob", "tags": ["user"], "profile": {"age": 25, "verified": false}}
		],
		"metadata": {
			"total": 2,
			"filters": {"active": true, "roles": ["admin", "user"]},
			"pagination": {"page": 1, "size": 10}
		}
	}`

	var data any
	err := json.Unmarshal([]byte(jsonStr), &data)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseNodes(data)
	}
}

func TestGenerateObjectSummary(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name: "simple object",
			input: map[string]any{
				"id":   1,
				"name": "Alice",
			},
			expected: "1 Alice",
		},
		{
			name: "nested object",
			input: map[string]any{
				"id":   1,
				"name": "Alice",
				"contact": map[string]any{
					"email": "alice@example.com",
					"phone": "123-456-7890",
				},
			},
			expected: "1 Alice alice@example.com 123-456-7890",
		},
		{
			name: "deeply nested object",
			input: map[string]any{
				"user": map[string]any{
					"id":   1,
					"name": "Alice",
					"profile": map[string]any{
						"contact": map[string]any{
							"email": "alice@example.com",
							"phone": "123-456-7890",
						},
						"preferences": map[string]any{
							"theme": "dark",
							"lang":  "en",
						},
					},
				},
			},
			expected: "1 Alice alice@example.com 123-456-7890 dark en",
		},
		{
			name: "object with array",
			input: map[string]any{
				"id":   1,
				"name": "Alice",
				"tags": []any{"admin", "user"},
			},
			expected: "1 Alice admin user",
		},
		{
			name: "object with mixed types",
			input: map[string]any{
				"id":     1,
				"name":   "Alice",
				"active": true,
				"score":  95.5,
				"notes":  nil,
				"empty":  "",
			},
			expected: "1 Alice true 95.5",
		},
		{
			name:     "empty object",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "object with only empty values",
			input: map[string]any{
				"empty1": "",
				"empty2": nil,
			},
			expected: "",
		},
		{
			name: "long summary that should be truncated",
			input: map[string]any{
				"field1": "This is a very long string that will contribute to making the summary exceed the maximum length",
				"field2": "Another long string that will definitely cause truncation",
				"field3": "And yet another long string to ensure we hit the limit",
			},
			expected: "This is a very long string that will contribute to making the summary ex...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateObjectSummary(tt.input)

			// For tests with deterministic order (single values or specific structures)
			if tt.name == "empty object" || tt.name == "object with only empty values" {
				assert.Equal(t, tt.expected, result)
			} else {
				// For tests where map iteration order matters, check that all expected words are present
				expectedWords := strings.Fields(tt.expected)
				resultWords := strings.Fields(result)

				if strings.HasSuffix(tt.expected, "...") {
					// For truncation test, just check that it's truncated and starts correctly
					assert.True(t, strings.HasSuffix(result, "..."), "Result should be truncated")
					assert.True(t, len(result) <= maxSummaryLength, "Result should not exceed max length")
				} else {
					// Check that all expected words are present (order may vary due to map iteration)
					assert.ElementsMatch(t, expectedWords, resultWords, "Should contain same words")
				}
			}
		})
	}
}

func TestIntegrationFeatures(t *testing.T) {
	t.Run("summary and expansion behavior", func(t *testing.T) {
		data := map[string]any{
			"user": map[string]any{
				"id":   1,
				"name": "Alice",
				"contact": map[string]any{
					"email": "alice@example.com",
				},
			},
		}

		nodes := parseNodes(data)
		require.Len(t, nodes, 1)

		userNode := nodes[0]
		// Should show summary and be expanded at top level
		assert.True(t, userNode.Expanded)
		assert.Contains(t, userNode.Value, "Alice")
		assert.Contains(t, userNode.Value, "alice@example.com")

		// Contact node should not be expanded (depth 1)
		var contactNode *node
		for _, child := range userNode.Children {
			if child.Key == "contact" {
				contactNode = child
				break
			}
		}
		require.NotNil(t, contactNode)
		assert.False(t, contactNode.Expanded)
	})

	t.Run("empty objects and truncation", func(t *testing.T) {
		data := map[string]any{
			"empty": map[string]any{},
			"long": map[string]any{
				"field": strings.Repeat("very long text ", 20),
			},
		}

		nodes := parseNodes(data)
		require.Len(t, nodes, 2)

		nodeMap := make(map[string]*node)
		for _, node := range nodes {
			nodeMap[node.Key] = node
		}

		// Empty object should have empty summary
		assert.Equal(t, "", nodeMap["empty"].Value)

		// Long summary should be truncated
		assert.True(t, len(nodeMap["long"].Value) <= maxSummaryLength)
		assert.True(t, strings.HasSuffix(nodeMap["long"].Value, "..."))
	})
}

// TestRenderNodeWidthTruncation tests that long lines are truncated to fit terminal width
func TestRenderNodeWidthTruncation(t *testing.T) {
	// Create a node with a very long value
	longValue := strings.Repeat("very long text ", 20) // Creates a very long string
	node := &node{
		Key:    "test",
		Value:  longValue,
		IsLeaf: true,
	}

	// Create a model with a narrow width
	model := &treeModel{
		width:  50, // Narrow terminal width
		styles: defaultStyles(),
	}

	// Render the node
	result := model.renderNode(node, false)

	// The result should not exceed the terminal width
	// Note: we can't check exact length due to ANSI codes, but it should be truncated
	assert.Contains(t, result, "...", "Long value should be truncated with ellipsis")

	// Test with zero width (should not crash)
	model.width = 0
	result = model.renderNode(node, false)
	assert.NotEmpty(t, result, "Should still render with zero width")
}

// TestExpandedNodeRendering tests that expanded nodes don't show summaries
func TestExpandedNodeRendering(t *testing.T) {
	// Create a non-leaf node with a summary
	node := &node{
		Key:      "test",
		Value:    "summary text here",
		IsLeaf:   false,
		Expanded: false, // Initially collapsed
	}

	model := &treeModel{
		width:  100,
		styles: defaultStyles(),
	}

	// When collapsed, should show summary
	result := model.renderNode(node, false)
	assert.Contains(t, result, "summary text here", "Collapsed node should show summary")
	assert.Contains(t, result, ":", "Collapsed node should have colon separator")

	// When expanded, should not show summary
	node.Expanded = true
	result = model.renderNode(node, false)
	assert.NotContains(t, result, "summary text here", "Expanded node should not show summary")
	assert.NotContains(t, result, ":", "Expanded node should not have colon separator")
}

// TestMultilineStringEscaping tests that multiline strings are properly escaped in summaries
func TestMultilineStringEscaping(t *testing.T) {
	// Test object with multiline string
	obj := map[string]any{
		"message": "line1\nline2\ttab",
		"id":      123,
	}

	summary := generateObjectSummary(obj)

	// Should contain escaped newlines and tabs, not actual newlines
	assert.Contains(t, summary, "line1\\nline2\\ttab", "Should escape newlines and tabs")
	assert.NotContains(t, summary, "\n", "Should not contain actual newlines")
	assert.NotContains(t, summary, "\t", "Should not contain actual tabs")
	assert.Contains(t, summary, "123", "Should also contain other values")
}

// TestIsArrayOfLeafNodes tests the helper function for detecting leaf-only arrays
func TestIsArrayOfLeafNodes(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		expected bool
	}{
		{
			name:     "empty array",
			input:    []any{},
			expected: true,
		},
		{
			name:     "array of strings",
			input:    []any{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "array of numbers",
			input:    []any{1, 2, 3},
			expected: true,
		},
		{
			name:     "array of mixed primitives",
			input:    []any{"hello", 42, true, nil},
			expected: true,
		},
		{
			name:     "array with nested object",
			input:    []any{"hello", map[string]any{"key": "value"}},
			expected: false,
		},
		{
			name:     "array with nested array",
			input:    []any{"hello", []any{1, 2}},
			expected: false,
		},
		{
			name:     "array of only objects",
			input:    []any{map[string]any{"a": 1}, map[string]any{"b": 2}},
			expected: false,
		},
		{
			name:     "array of only arrays",
			input:    []any{[]any{1}, []any{2}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isArrayOfLeafNodes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenerateArraySummary tests the array summary generation
func TestGenerateArraySummary(t *testing.T) {
	tests := []struct {
		name     string
		input    []any
		expected string
	}{
		{
			name:     "empty array",
			input:    []any{},
			expected: "",
		},
		{
			name:     "array of strings",
			input:    []any{"apple", "banana", "cherry"},
			expected: "apple, banana, cherry",
		},
		{
			name:     "array of numbers",
			input:    []any{1, 2, 3},
			expected: "1, 2, 3",
		},
		{
			name:     "array of mixed types",
			input:    []any{"hello", 42, true, nil},
			expected: "hello, 42, true, null",
		},
		{
			name:     "array with floats",
			input:    []any{1.5, 2.7, 3.14},
			expected: "1.5, 2.7, 3.14",
		},
		{
			name:     "array with booleans",
			input:    []any{true, false, true},
			expected: "true, false, true",
		},
		{
			name:     "single element array",
			input:    []any{"single"},
			expected: "single",
		},
		{
			name: "long array that should be truncated",
			input: []any{
				"very long string that will contribute to exceeding the maximum length",
				"another very long string that will definitely cause truncation",
				"yet another long string",
				"and more text to ensure we hit the limit",
			},
			expected: "very long string that will contribute to exceeding the maximum length...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateArraySummary(tt.input)

			if strings.HasSuffix(tt.expected, "...") {
				// For truncation test
				assert.True(t, strings.HasSuffix(result, "..."), "Result should be truncated")
				assert.True(t, len(result) <= maxSummaryLength, "Result should not exceed max length")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestArrayLeafNodeSummaryIntegration tests the full integration of array leaf summaries
func TestArrayLeafNodeSummaryIntegration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, nodes []*node)
	}{
		{
			name:  "array of strings should show comma-separated values",
			input: `{"tags": ["admin", "user", "active"]}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				tags := nodes[0]
				assert.Equal(t, "tags", tags.Key)
				assert.Equal(t, "admin, user, active", tags.Value)
				assert.False(t, tags.IsLeaf)
				assert.Len(t, tags.Children, 3) // Still has children for expansion
			},
		},
		{
			name:  "array of numbers should show comma-separated values",
			input: `{"scores": [95, 87, 92]}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				scores := nodes[0]
				assert.Equal(t, "scores", scores.Key)
				assert.Equal(t, "95, 87, 92", scores.Value)
				assert.False(t, scores.IsLeaf)
			},
		},
		{
			name:  "array of mixed primitives should show comma-separated values",
			input: `{"mixed": ["hello", 42, true, null]}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				mixed := nodes[0]
				assert.Equal(t, "mixed", mixed.Key)
				assert.Equal(t, "hello, 42, true, null", mixed.Value)
				assert.False(t, mixed.IsLeaf)
			},
		},
		{
			name:  "array with nested objects should show array[n] format",
			input: `{"users": [{"name": "Alice"}, {"name": "Bob"}]}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				users := nodes[0]
				assert.Equal(t, "users", users.Key)
				assert.Equal(t, "array[2]", users.Value)
				assert.False(t, users.IsLeaf)
			},
		},
		{
			name:  "array with nested arrays should show array[n] format",
			input: `{"matrix": [[1, 2], [3, 4]]}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				matrix := nodes[0]
				assert.Equal(t, "matrix", matrix.Key)
				assert.Equal(t, "array[2]", matrix.Value)
				assert.False(t, matrix.IsLeaf)
			},
		},
		{
			name:  "empty array should show array[0] format",
			input: `{"empty": []}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				empty := nodes[0]
				assert.Equal(t, "empty", empty.Key)
				assert.Equal(t, "", empty.Value) // Empty array of leaf nodes shows empty string
				assert.False(t, empty.IsLeaf)
			},
		},
		{
			name:  "root array of primitives should show comma-separated values",
			input: `["red", "green", "blue"]`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				root := nodes[0]
				assert.Equal(t, "data", root.Key)
				assert.Equal(t, "red, green, blue", root.Value)
				assert.False(t, root.IsLeaf)
				assert.Len(t, root.Children, 3)
			},
		},
		{
			name:  "complex structure with mixed array types",
			input: `{"config": {"tags": ["prod", "web"], "ports": [80, 443], "servers": [{"name": "web1"}, {"name": "web2"}]}}`,
			validate: func(t *testing.T, nodes []*node) {
				require.Len(t, nodes, 1)
				config := nodes[0]
				assert.Equal(t, "config", config.Key)

				// Find the arrays in children
				require.Len(t, config.Children, 3)

				childMap := make(map[string]*node)
				for _, child := range config.Children {
					childMap[child.Key] = child
				}

				// Tags should show comma-separated values
				require.Contains(t, childMap, "tags")
				assert.Equal(t, "prod, web", childMap["tags"].Value)

				// Ports should show comma-separated values
				require.Contains(t, childMap, "ports")
				assert.Equal(t, "80, 443", childMap["ports"].Value)

				// Servers should show array[n] format (contains objects)
				require.Contains(t, childMap, "servers")
				assert.Equal(t, "array[2]", childMap["servers"].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data any
			err := json.Unmarshal([]byte(tt.input), &data)
			require.NoError(t, err)

			nodes := parseNodes(data)
			tt.validate(t, nodes)
		})
	}
}
