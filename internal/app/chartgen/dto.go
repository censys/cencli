package chartgen

import (
	"github.com/censys/cencli/internal/app/chartgen/prompts"
)

// Params contains parameters for chart generation.
type Params struct {
	// Query is the Censys query string used to generate the data.
	Query string
	// Field is the aggregation field.
	Field string
	// ChartType is the type of chart to generate (e.g., "geomap", "pie", "bar").
	ChartType string
	// Buckets contains the aggregation data.
	Buckets []prompts.Bucket
	// TotalCount is the total count across all buckets.
	TotalCount uint64
	// OtherCount is the count of items not in the top buckets.
	OtherCount uint64
	// NumImages is the number of images to generate.
	NumImages int
}

// Result contains the generated chart images.
type Result struct {
	// Images contains the generated PNG image bytes.
	Images [][]byte
	// Prompt is the prompt used for generation.
	Prompt string
}
