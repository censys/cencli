package golden

import (
	_ "embed"
)

var (
	//go:embed view_help.out
	ViewHelpStdout []byte
	//go:embed aggregate_help.out
	AggregateHelpStdout []byte
	//go:embed search_help.out
	SearchHelpStdout []byte
	//go:embed censeye_help.out
	CenseyeHelpStdout []byte
	//go:embed history_help.out
	HistoryHelpStdout []byte
	//go:embed root.out
	RootStdout []byte
)
