package censeye

import (
	"sort"

	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/censys-sdk-go/models/components"
)

type InvestigateHostResult struct {
	Entries []ReportEntry
	Meta    *responsemeta.ResponseMeta
}

// reportEntry represents a single rule and its analysis results.
type ReportEntry struct {
	Count       int64  `json:"count"`
	Query       string `json:"query"`
	Interesting bool   `json:"interesting"`
	SearchURL   string `json:"search_url,omitempty"`
}

type fieldValuePair struct {
	// The field to match
	Field string
	// The value to match
	Value string
}

type countCondition struct {
	// Field-value pairs to count matches for. Must target fields from the same nested object.
	FieldValuePairs []fieldValuePair
}

// valueCountsResult is the typed result of a value counts call.
type valueCountsResult struct {
	Meta            *responsemeta.ResponseMeta
	AndCountResults []float64
}

func (c *countCondition) marshal() components.CountCondition {
	pairs := make([]components.FieldValuePair, 0, len(c.FieldValuePairs))
	for _, pair := range c.FieldValuePairs {
		pairs = append(pairs, components.FieldValuePair{
			Field: pair.Field,
			Value: pair.Value,
		})
	}
	return components.CountCondition{
		FieldValuePairs: pairs,
	}
}

func marshalCountConditionSlice(countConditions []countCondition) []components.CountCondition {
	marshaledCountConditions := make([]components.CountCondition, 0, len(countConditions))
	for _, condition := range countConditions {
		marshaledCountConditions = append(marshaledCountConditions, condition.marshal())
	}
	return marshaledCountConditions
}

// buildReportEntries builds the report entries from the rules and counts.
// Filters out entries with counts less than or equal to 1.
func buildReportEntries(
	rules [][]fieldValuePair,
	counts []float64,
	rarityMin,
	rarityMax uint64,
) []ReportEntry {
	entries := make([]ReportEntry, 0, len(rules))
	for i, rule := range rules {
		// guard against out-of-bounds access if API returns fewer counts than rules
		var count uint64
		if i < len(counts) {
			count = uint64(counts[i])
		}
		if count > 1 {
			cenqlQuery := toCenqlQuery(rule)
			entry := ReportEntry{
				Count:       int64(count),
				Query:       cenqlQuery,
				Interesting: count >= rarityMin && count <= rarityMax,
				SearchURL:   toSearchURL(cenqlQuery),
			}
			entries = append(entries, entry)
		}
	}
	// sort by count descending
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Query < entries[j].Query
		}
		return entries[i].Count > entries[j].Count
	})
	return entries
}
