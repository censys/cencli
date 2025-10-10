package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/samber/mo"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/threathunting_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys ThreatHuntingClient
type ThreatHuntingClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/threathunting#gethostobservationswithcertificate
	GetHostObservationsWithCertificate(
		ctx context.Context,
		orgID mo.Option[string],
		certificateID string,
		startTime mo.Option[time.Time],
		endTime mo.Option[time.Time],
		port mo.Option[int],
		protocol mo.Option[string],
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.HostObservationResponse], ClientError)

	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/threathunting#valuecounts
	GetValueCounts(
		ctx context.Context,
		orgID mo.Option[string],
		query mo.Option[string],
		andCountConditions []components.CountCondition,
	) (Result[components.ValueCountsResponse], ClientError)
}

type threatHuntingSDK struct {
	*censysSDK
}

var _ ThreatHuntingClient = &threatHuntingSDK{}

func newThreatHuntingSDK(censysSDK *censysSDK) *threatHuntingSDK {
	return &threatHuntingSDK{censysSDK: censysSDK}
}

func (t *threatHuntingSDK) GetHostObservationsWithCertificate(
	ctx context.Context,
	orgID mo.Option[string],
	certificateID string,
	startTime mo.Option[time.Time],
	endTime mo.Option[time.Time],
	port mo.Option[int],
	protocol mo.Option[string],
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.HostObservationResponse], ClientError) {
	start := time.Now()
	var res *operations.V3ThreathuntingGetHostObservationsWithCertificateResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		// Build RFC3339 strings for times per SDK request model
		var startStr, endStr *string
		if startTime.IsPresent() {
			s := startTime.MustGet().UTC().Format(time.RFC3339)
			startStr = &s
		}
		if endTime.IsPresent() {
			e := endTime.MustGet().UTC().Format(time.RFC3339)
			endStr = &e
		}
		req := operations.V3ThreathuntingGetHostObservationsWithCertificateRequest{
			OrganizationID: orgID.ToPointer(),
			CertificateID:  certificateID,
			StartTime:      startStr,
			EndTime:        endStr,
			Port:           port.ToPointer(),
			Protocol:       protocol.ToPointer(),
			PageToken:      pageToken.ToPointer(),
		}
		if pageSize.IsPresent() {
			ps := int(pageSize.MustGet())
			req.PageSize = &ps
		}
		var err error
		res, err = t.censysSDK.client.ThreatHunting.GetHostObservationsWithCertificate(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.HostObservationResponse]{}
		return zero, err
	}
	observation := res.GetResponseEnvelopeHostObservationResponse().GetResult()
	return Result[components.HostObservationResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     observation,
	}, nil
}

func (c *threatHuntingSDK) GetValueCounts(
	ctx context.Context,
	orgID mo.Option[string],
	query mo.Option[string],
	andCountConditions []components.CountCondition,
) (Result[components.ValueCountsResponse], ClientError) {
	start := time.Now()
	var res *operations.V3ThreathuntingValueCountsResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.ThreatHunting.ValueCounts(ctx, operations.V3ThreathuntingValueCountsRequest{
			OrganizationID: orgID.ToPointer(),
			SearchValueCountsInputBody: components.SearchValueCountsInputBody{
				Query:              query.ToPointer(),
				AndCountConditions: andCountConditions,
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.ValueCountsResponse]{}
		return zero, err
	}
	valueCountsResponse := res.GetResponseEnvelopeValueCountsResponse().GetResult()
	return Result[components.ValueCountsResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     valueCountsResponse,
	}, nil
}
