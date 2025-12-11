package prompts

import (
	"fmt"
	"strings"
)

// Bucket represents a single aggregation bucket with a key and count.
type Bucket struct {
	Key   string
	Count uint64
}

// PromptBuilder builds prompts for chart generation from aggregation data.
type PromptBuilder struct {
	prompt     string
	data       string
	style      string
	query      string
	field      string
	totalCount uint64
	otherCount uint64
}

// New creates a new PromptBuilder from aggregation buckets.
func New(buckets []Bucket, totalCount, otherCount uint64) *PromptBuilder {
	var dataStr string
	if len(buckets) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Total Count, %d\n", totalCount))
		sb.WriteString(fmt.Sprintf("Other Count, %d\n", otherCount))
		for _, bucket := range buckets {
			sb.WriteString(fmt.Sprintf("%s, %d\n", bucket.Key, bucket.Count))
		}
		dataStr = sb.String()
	}
	return &PromptBuilder{
		prompt:     BasePrompt,
		data:       dataStr,
		totalCount: totalCount,
		otherCount: otherCount,
	}
}

// WithQuery sets the query string for the prompt.
func (p *PromptBuilder) WithQuery(query string) *PromptBuilder {
	p.query = query
	return p
}

// WithField sets the aggregation field for the prompt.
func (p *PromptBuilder) WithField(field string) *PromptBuilder {
	p.field = field
	return p
}

// WithChartType sets the chart type for the prompt.
func (p *PromptBuilder) WithChartType(chartType string) *PromptBuilder {
	if style, ok := ChartPrompts[chartType]; ok {
		p.style = style
	}
	return p
}

// Build constructs the final prompt string.
func (p *PromptBuilder) Build() string {
	metadata := ""
	if p.query != "" {
		metadata += fmt.Sprintf("\nCensys Query: %s", p.query)
	}
	if p.field != "" {
		metadata += fmt.Sprintf("\nAggregation Field: %s", p.field)
	}
	return fmt.Sprintf("%s\n%s\n%s\nDATA:\n%s", p.prompt, p.style, metadata, p.data)
}
