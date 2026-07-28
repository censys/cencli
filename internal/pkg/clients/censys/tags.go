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

// CreateTagRequest bundles the fields for CreateTag.
type CreateTagRequest struct {
	OrgID       mo.Option[string]
	Name        string
	Description mo.Option[string]
	Privacy     string
}

// UpdateTagRequest bundles the fields for UpdateTag. Fields are optional; only
// present options are sent.
type UpdateTagRequest struct {
	OrgID       mo.Option[string]
	TagID       string
	Name        mo.Option[string]
	Description mo.Option[string]
	Privacy     mo.Option[string]
}

// CreateTagAssignmentRequest bundles the fields for CreateTagAssignment. TagID
// is the resolved tag UUID.
type CreateTagAssignmentRequest struct {
	OrgID   mo.Option[string]
	TagID   string
	AssetID string
}

// ListTagAssignmentsRequest bundles the query parameters for ListTagAssignments.
// TagID is the resolved tag UUID; only present options are sent.
type ListTagAssignmentsRequest struct {
	OrgID         mo.Option[string]
	TagID         string
	AssetID       mo.Option[string]
	AssetType     mo.Option[string]
	CreatedBy     mo.Option[string]
	CreatedBefore mo.Option[time.Time]
	CreatedAfter  mo.Option[time.Time]
	OrderBy       mo.Option[string]
	PageSize      mo.Option[int64]
	PageToken     mo.Option[string]
}

// BulkCreateTagAssignmentsRequest bundles the fields for
// BulkCreateTagAssignments. TagID is the resolved tag UUID; an absent MaxAssets
// leaves the cap to the plan's tag asset limit.
type BulkCreateTagAssignmentsRequest struct {
	OrgID     mo.Option[string]
	TagID     string
	Query     string
	MaxAssets mo.Option[int64]
}

// ListTagOperationsRequest bundles the query parameters for ListTagOperations.
// TagID is a resolved tag UUID, or "-" to list operations across every tag in
// the organization; only present options are sent.
type ListTagOperationsRequest struct {
	OrgID     mo.Option[string]
	TagID     string
	Type      mo.Option[string]
	Status    mo.Option[string]
	OrderBy   mo.Option[string]
	PageSize  mo.Option[int64]
	PageToken mo.Option[string]
}

//go:generate mockgen -destination=../../../../gen/client/mocks/tags_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys TagsClient
type TagsClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#listtags
	ListTags(ctx context.Context, req ListTagsRequest) (Result[components.TagsList], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#gettag
	GetTag(
		ctx context.Context,
		orgID mo.Option[string],
		tagID string,
	) (Result[components.Tag], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#createtag
	CreateTag(ctx context.Context, req CreateTagRequest) (Result[components.Tag], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#updatetag
	UpdateTag(ctx context.Context, req UpdateTagRequest) (Result[components.Tag], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#deletetag
	//
	// DeleteTag returns only response metadata; the endpoint has no body.
	DeleteTag(ctx context.Context, orgID mo.Option[string], tagID string) (Metadata, ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#createtagassignment
	CreateTagAssignment(ctx context.Context, req CreateTagAssignmentRequest) (Result[components.TagAssignment], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#bulkcreatetagassignments
	//
	// BulkCreateTagAssignments starts an asynchronous job and returns the
	// operation tracking it, not the assignments themselves.
	BulkCreateTagAssignments(ctx context.Context, req BulkCreateTagAssignmentsRequest) (Result[components.TagOperation], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#listtagassignments
	ListTagAssignments(ctx context.Context, req ListTagAssignmentsRequest) (Result[components.TagAssignmentsList], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#deletetagassignment
	//
	// DeleteTagAssignment returns only response metadata; the endpoint has no body.
	DeleteTagAssignment(ctx context.Context, orgID mo.Option[string], tagID, assignmentID string) (Metadata, ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#listtagoperations
	ListTagOperations(ctx context.Context, req ListTagOperationsRequest) (Result[components.TagOperationsList], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/tagsandcomments#gettagoperation
	GetTagOperation(
		ctx context.Context,
		orgID mo.Option[string],
		tagID, operationID string,
	) (Result[components.TagOperation], ClientError)
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

func (t *tagsSDK) GetTag(
	ctx context.Context,
	orgID mo.Option[string],
	tagID string,
) (Result[components.Tag], ClientError) {
	start := time.Now()
	var res *operations.V3TagsGetTagResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3TagsGetTagRequest{
			OrganizationID: orgID.ToPointer(),
			TagID:          tagID,
		}
		res, err = t.censysSDK.client.TagsAndComments.GetTag(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Tag]{}
		return zero, err
	}
	tag := res.GetResponseEnvelopeTag().GetResult()
	return Result[components.Tag]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     tag,
	}, nil
}

func (t *tagsSDK) ListTagOperations(
	ctx context.Context,
	req ListTagOperationsRequest,
) (Result[components.TagOperationsList], ClientError) {
	start := time.Now()
	var res *operations.V3TagsListOperationsResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsListOperationsRequest{
			OrganizationID: req.OrgID.ToPointer(),
			TagID:          req.TagID,
			PageToken:      req.PageToken.ToPointer(),
		}
		if req.PageSize.IsPresent() {
			ps := int(req.PageSize.MustGet())
			sdkReq.PageSize = &ps
		}
		// The SDK names the operation filter enums bare Type and Status; they are
		// distinct from the components.TagOperation* enums on the response.
		if req.Type.IsPresent() {
			ty := operations.Type(req.Type.MustGet())
			sdkReq.Type = &ty
		}
		if req.Status.IsPresent() {
			st := operations.Status(req.Status.MustGet())
			sdkReq.Status = &st
		}
		if req.OrderBy.IsPresent() {
			ob := operations.V3TagsListOperationsQueryParamOrderBy(req.OrderBy.MustGet())
			sdkReq.OrderBy = &ob
		}
		res, err = t.censysSDK.client.TagsAndComments.ListTagOperations(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagOperationsList]{}
		return zero, err
	}
	opsList := res.GetResponseEnvelopeTagOperationsList().GetResult()
	return Result[components.TagOperationsList]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     opsList,
	}, nil
}

func (t *tagsSDK) GetTagOperation(
	ctx context.Context,
	orgID mo.Option[string],
	tagID, operationID string,
) (Result[components.TagOperation], ClientError) {
	start := time.Now()
	var res *operations.V3TagsGetOperationResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3TagsGetOperationRequest{
			OrganizationID: orgID.ToPointer(),
			TagID:          tagID,
			OperationID:    operationID,
		}
		res, err = t.censysSDK.client.TagsAndComments.GetTagOperation(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagOperation]{}
		return zero, err
	}
	op := res.GetResponseEnvelopeTagOperation().GetResult()
	return Result[components.TagOperation]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     op,
	}, nil
}

func (t *tagsSDK) DeleteTag(
	ctx context.Context,
	orgID mo.Option[string],
	tagID string,
) (Metadata, ClientError) {
	start := time.Now()
	var res *operations.V3TagsDeleteTagResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3TagsDeleteTagRequest{
			OrganizationID: orgID.ToPointer(),
			TagID:          tagID,
		}
		res, err = t.censysSDK.client.TagsAndComments.DeleteTag(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		return Metadata{}, err
	}
	// The delete endpoint returns no body, only response metadata.
	return buildResponseMetadata(res, latency, attempts), nil
}

func (t *tagsSDK) CreateTag(
	ctx context.Context,
	req CreateTagRequest,
) (Result[components.Tag], ClientError) {
	start := time.Now()
	var res *operations.V3TagsCreateTagResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsCreateTagRequest{
			OrganizationID: req.OrgID.ToPointer(),
			CreateTagInputBody: components.CreateTagInputBody{
				Name:        req.Name,
				Description: req.Description.ToPointer(),
				Privacy:     components.CreateTagInputBodyPrivacy(req.Privacy),
			},
		}
		res, err = t.censysSDK.client.TagsAndComments.CreateTag(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Tag]{}
		return zero, err
	}
	tag := res.GetResponseEnvelopeTag().GetResult()
	return Result[components.Tag]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     tag,
	}, nil
}

func (t *tagsSDK) CreateTagAssignment(
	ctx context.Context,
	req CreateTagAssignmentRequest,
) (Result[components.TagAssignment], ClientError) {
	start := time.Now()
	var res *operations.V3TagsCreateAssignmentResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsCreateAssignmentRequest{
			OrganizationID: req.OrgID.ToPointer(),
			TagID:          req.TagID,
			CreateTagAssignmentInputBody: components.CreateTagAssignmentInputBody{
				AssetID: req.AssetID,
			},
		}
		res, err = t.censysSDK.client.TagsAndComments.CreateTagAssignment(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagAssignment]{}
		return zero, err
	}
	assignment := res.GetResponseEnvelopeTagAssignment().GetResult()
	return Result[components.TagAssignment]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     assignment,
	}, nil
}

func (t *tagsSDK) BulkCreateTagAssignments(
	ctx context.Context,
	req BulkCreateTagAssignmentsRequest,
) (Result[components.TagOperation], ClientError) {
	start := time.Now()
	var res *operations.V3TagsBulkCreateAssignmentsResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsBulkCreateAssignmentsRequest{
			OrganizationID: req.OrgID.ToPointer(),
			TagID:          req.TagID,
			BulkCreateTagAssignmentsInputBody: components.BulkCreateTagAssignmentsInputBody{
				Query:     req.Query,
				MaxAssets: req.MaxAssets.ToPointer(),
			},
		}
		res, err = t.censysSDK.client.TagsAndComments.BulkCreateTagAssignments(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagOperation]{}
		return zero, err
	}
	operation := res.GetResponseEnvelopeTagOperation().GetResult()
	return Result[components.TagOperation]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     operation,
	}, nil
}

func (t *tagsSDK) ListTagAssignments(
	ctx context.Context,
	req ListTagAssignmentsRequest,
) (Result[components.TagAssignmentsList], ClientError) {
	start := time.Now()
	var res *operations.V3TagsListAssignmentsResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3TagsListAssignmentsRequest{
			OrganizationID: req.OrgID.ToPointer(),
			TagID:          req.TagID,
			AssetID:        req.AssetID.ToPointer(),
			CreatedBy:      req.CreatedBy.ToPointer(),
			CreatedBefore:  req.CreatedBefore.ToPointer(),
			CreatedAfter:   req.CreatedAfter.ToPointer(),
			PageToken:      req.PageToken.ToPointer(),
		}
		if req.PageSize.IsPresent() {
			ps := int(req.PageSize.MustGet())
			sdkReq.PageSize = &ps
		}
		if req.AssetType.IsPresent() {
			at := operations.AssetType(req.AssetType.MustGet())
			sdkReq.AssetType = &at
		}
		if req.OrderBy.IsPresent() {
			ob := operations.V3TagsListAssignmentsQueryParamOrderBy(req.OrderBy.MustGet())
			sdkReq.OrderBy = &ob
		}
		res, err = t.censysSDK.client.TagsAndComments.ListTagAssignments(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TagAssignmentsList]{}
		return zero, err
	}
	assignments := res.GetResponseEnvelopeTagAssignmentsList().GetResult()
	return Result[components.TagAssignmentsList]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     assignments,
	}, nil
}

func (t *tagsSDK) DeleteTagAssignment(
	ctx context.Context,
	orgID mo.Option[string],
	tagID, assignmentID string,
) (Metadata, ClientError) {
	start := time.Now()
	var res *operations.V3TagsDeleteAssignmentResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3TagsDeleteAssignmentRequest{
			OrganizationID: orgID.ToPointer(),
			TagID:          tagID,
			AssignmentID:   assignmentID,
		}
		res, err = t.censysSDK.client.TagsAndComments.DeleteTagAssignment(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		return Metadata{}, err
	}
	// The delete endpoint returns no body, only response metadata.
	return buildResponseMetadata(res, latency, attempts), nil
}

func (t *tagsSDK) UpdateTag(
	ctx context.Context,
	req UpdateTagRequest,
) (Result[components.Tag], ClientError) {
	start := time.Now()
	var res *operations.V3TagsUpdateTagResponse
	err, attempts := t.executeWithRetry(ctx, func() ClientError {
		var err error
		body := components.UpdateTagInputBody{
			Name:        req.Name.ToPointer(),
			Description: req.Description.ToPointer(),
		}
		if req.Privacy.IsPresent() {
			p := components.UpdateTagInputBodyPrivacy(req.Privacy.MustGet())
			body.Privacy = &p
		}
		sdkReq := operations.V3TagsUpdateTagRequest{
			OrganizationID:     req.OrgID.ToPointer(),
			TagID:              req.TagID,
			UpdateTagInputBody: body,
		}
		res, err = t.censysSDK.client.TagsAndComments.UpdateTag(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Tag]{}
		return zero, err
	}
	tag := res.GetResponseEnvelopeTag().GetResult()
	return Result[components.Tag]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     tag,
	}, nil
}
