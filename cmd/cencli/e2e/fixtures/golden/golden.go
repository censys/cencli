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
	//go:embed credits_help.out
	CreditsHelpStdout []byte
	//go:embed org_details_help.out
	OrgDetailsHelpStdout []byte
	//go:embed org_members_help.out
	OrgMembersHelpStdout []byte
	//go:embed org_credits_help.out
	OrgCreditsHelpStdout []byte
	//go:embed org_help.out
	OrgHelpStdout []byte
)
