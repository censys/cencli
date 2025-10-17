package config

// SearchConfig contains defaults for search pagination.
type SearchConfig struct {
	// PageSize sets the default number of results per page for search.
	// Must be >= 1.
	PageSize int64 `yaml:"page-size" mapstructure:"page-size" doc:"Default number of results per page (must be >= 1)"`
	// MaxPages limits the number of pages fetched. Set to -1 for unlimited.
	// 0 is invalid and will be rejected.
	MaxPages int64 `yaml:"max-pages" mapstructure:"max-pages" doc:"Number of pages to fetch (max is 100)"`
}

var defaultSearchConfig = SearchConfig{
	PageSize: 100,
	MaxPages: 1,
}
