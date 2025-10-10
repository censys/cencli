package censeye

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/censeye/mocks/censeyeservice_mock.go -package=mocks -mock_names Service=MockCenseyeService . Service

// Service provides censeye capabilities.
type Service interface {
	InvestigateHost(
		ctx context.Context,
		orgID mo.Option[identifiers.OrganizationID],
		host *assets.Host,
		rarityMin uint64,
		rarityMax uint64,
	) (InvestigateHostResult, cenclierrors.CencliError)
}

type censeyeService struct {
	client client.Client
}

func New(client client.Client) Service { return &censeyeService{client: client} }

func (s *censeyeService) InvestigateHost(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	host *assets.Host,
	rarityMin uint64,
	rarityMax uint64,
) (InvestigateHostResult, cenclierrors.CencliError) {
	// compile rules from host data
	progress.ReportMessage(ctx, progress.StageProcess, "Compiling detection rules from host data...")
	rules, compileErr := compileRulesForHost(host, &defaultCenseyeConfig)
	if compileErr != nil {
		return InvestigateHostResult{}, newCompileRulesError(compileErr)
	}

	// apply filters
	progress.ReportMessage(ctx, progress.StageProcess, fmt.Sprintf("Applying filters (%d rules found)...", len(rules)))
	filteredRules := applyFilters(rules, &defaultCenseyeConfig)

	// prepare count conditions
	countConditions := make([]countCondition, 0, len(filteredRules))
	for _, rule := range filteredRules {
		countConditions = append(countConditions, countCondition{FieldValuePairs: rule})
	}

	// get value counts from threat hunting service
	progress.ReportMessage(ctx, progress.StageProcess, fmt.Sprintf("Querying threat hunting service (%d conditions)...", len(filteredRules)))
	result, err := s.getValueCounts(ctx, orgID, countConditions, mo.None[string]())
	if err != nil {
		return InvestigateHostResult{}, err
	}

	// build report entries with configured rarity bounds
	progress.ReportMessage(ctx, progress.StageProcess, fmt.Sprintf("Analyzing rarity (bounds: %d-%d)...", rarityMin, rarityMax))
	entries := buildReportEntries(filteredRules, result.AndCountResults, rarityMin, rarityMax)
	return InvestigateHostResult{Entries: entries, Meta: result.Meta}, nil
}

func (s *censeyeService) getValueCounts(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	andCountConditions []countCondition,
	query mo.Option[string],
) (valueCountsResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(orgID)
	res, err := s.client.GetValueCounts(ctx, orgIDStr, query, marshalCountConditionSlice(andCountConditions))
	if err != nil {
		return valueCountsResult{}, err
	}
	return valueCountsResult{
		Meta:            responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts),
		AndCountResults: res.Data.GetAndCountResults(),
	}, nil
}
