package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/samber/mo"
)

// ListTagsRequest bundles the query parameters for ListTags.
type ListTagsRequest struct {
	OrgID     mo.Option[string]
	PageSize  mo.Option[int64]
	PageToken mo.Option[string]
	OrderBy   mo.Option[string]
	Name      mo.Option[string]
	CreatedBy mo.Option[string]
	Privacy   mo.Option[string]
}

//go:generate mockgen -destination=../../../../gen/client/mocks/tags_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys TagsClient
type TagsClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#listtags
	ListTags(ctx context.Context, req ListTagsRequest) (Result[components.TagsList], ClientError)
}

type tagsSDK struct {
	*censysSDK
}

var _ TagsClient = &tagsSDK{}

func newTagsSDK(censysSDK *censysSDK) *tagsSDK {
	return &tagsSDK{
		censysSDK: censysSDK,
	}
}

func (t *tagsSDK) ListTags(
	ctx context.Context,
	req ListTagsRequest,
) (Result[components.TagsList], ClientError) {
	start := time.Now()
	var res *operations.V3TagsListTagsResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsListTagsRequest{
			OrganizationID: req.OrgID.ToPointer(),
			PageToken:      req.PageToken.ToPointer(),
			Name:           req.Name.ToPointer(),
			CreatedBy:      req.CreatedBy.ToPointer(),
		}
		if req.PageSize.IsPresent() {
			ps := int(req.PageSize.MustGet())
			sdkReq.PageSize = &ps
		}
		if req.OrderBy.IsPresent() {
			ob := operations.V3TagsListTagsQueryParamOrderBy(req.OrderBy.MustGet())
			sdkReq.OrderBy = &ob
		}
		if req.Privacy.IsPresent() {
			p := operations.Privacy(req.Privacy.MustGet())
			sdkReq.Privacy = &p
		}
		res, err = t.censysSDK.client.TagsAndComments.ListTags(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagsList]{}
		return zero, err
	}
	tagsList := res.GetResponseEnvelopeTagsList().GetResult()
	return Result[components.TagsList]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     tagsList,
	}, nil
}
