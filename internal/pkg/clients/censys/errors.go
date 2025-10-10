package censys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

func NewClientError(err error) ClientError {
	var modelErr *sdkerrors.ErrorModel
	if errors.As(err, &modelErr) {
		return NewCensysClientStructuredError(modelErr)
	}
	var authErr *sdkerrors.AuthenticationError
	if errors.As(err, &authErr) {
		return NewCensysClientUnauthorizedError(authErr)
	}
	var genericErr *sdkerrors.SDKError
	if errors.As(err, &genericErr) {
		return NewCensysClientGenericError(genericErr)
	}
	if errors.Is(err, context.Canceled) {
		return wrapCencliError(cenclierrors.NewInterruptedError())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return wrapCencliError(cenclierrors.NewDeadlineExceededError())
	}
	return wrapCencliError(cenclierrors.NewCencliError(err))
}

type ClientError interface {
	cenclierrors.CencliError
	Status() string
	StatusCode() mo.Option[int64]
}

// clientErrorAdapter wraps any CencliError to make it ClientError-compatible.
// This allows non-client-specific errors (like interrupted, deadline exceeded,
// or unknown errors) to be used in client contexts.
type clientErrorAdapter struct {
	cenclierrors.CencliError
}

var _ ClientError = &clientErrorAdapter{}

func wrapCencliError(err cenclierrors.CencliError) ClientError {
	return &clientErrorAdapter{CencliError: err}
}

func (e *clientErrorAdapter) Status() string {
	return "unknown"
}

func (e *clientErrorAdapter) StatusCode() mo.Option[int64] {
	return mo.None[int64]()
}

type ClientStructuredError interface {
	ClientError
}

type errorDetail struct {
	location mo.Option[string]
	message  mo.Option[string]
	value    any
}

type censysClientError struct {
	detail   mo.Option[string]
	title    mo.Option[string]
	status   mo.Option[int64]
	errors   []errorDetail
	typ      mo.Option[string]
	instance mo.Option[string]
}

var _ ClientStructuredError = &censysClientError{}

func NewCensysClientStructuredError(err *sdkerrors.ErrorModel) ClientStructuredError {
	errors := make([]errorDetail, len(err.Errors))
	for i, err := range err.Errors {
		errors[i] = errorDetail{
			location: mo.PointerToOption(err.Location),
			message:  mo.PointerToOption(err.Message),
			value:    err.Value,
		}
	}
	return &censysClientError{
		detail:   mo.PointerToOption(err.Detail),
		title:    mo.PointerToOption(err.Title),
		status:   mo.PointerToOption(err.Status),
		errors:   errors,
		typ:      mo.PointerToOption(err.Type),
		instance: mo.PointerToOption(err.Instance),
	}
}

func (e *censysClientError) Error() string {
	type errStruct struct {
		Location *string `json:"location,omitempty"`
		Message  *string `json:"message,omitempty"`
		Value    any     `json:"value,omitempty"`
	}

	errs := make([]errStruct, 0, len(e.errors))
	for _, ed := range e.errors {
		errs = append(errs, errStruct{
			Location: ed.location.ToPointer(),
			Message:  ed.message.ToPointer(),
			Value:    ed.value,
		})
	}

	data := struct {
		Title    *string     `json:"title,omitempty"`
		Detail   *string     `json:"detail,omitempty"`
		Status   *int64      `json:"status,omitempty"`
		Type     *string     `json:"type,omitempty"`
		Instance *string     `json:"instance,omitempty"`
		Errors   []errStruct `json:"errors,omitempty"`
	}{
		Title:    e.title.ToPointer(),
		Detail:   e.detail.ToPointer(),
		Status:   e.status.ToPointer(),
		Type:     e.typ.ToPointer(),
		Instance: e.instance.ToPointer(),
		Errors:   errs,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to marshal censysClientError: %v", err)
	}
	return string(b)
}

func (e *censysClientError) Title() string {
	return "Error Returned from Censys API"
}

func (e *censysClientError) ShouldPrintUsage() bool {
	return false
}

func (e *censysClientError) Status() string {
	if e.status.IsPresent() {
		statusText := http.StatusText(int(e.status.MustGet()))
		if statusText != "" {
			return statusText
		}
	}
	return "unknown"
}

func (e *censysClientError) StatusCode() mo.Option[int64] {
	return e.status
}

type ClientUnauthorizedError interface {
	ClientError
}

type censysClientUnauthorizedError struct {
	code    mo.Option[int64]
	status  mo.Option[string]
	message mo.Option[string]
	reason  mo.Option[string]
}

var _ ClientUnauthorizedError = &censysClientUnauthorizedError{}

func NewCensysClientUnauthorizedError(err *sdkerrors.AuthenticationError) ClientUnauthorizedError {
	return &censysClientUnauthorizedError{
		code:    mo.PointerToOption(err.Error_.GetCode()),
		status:  mo.PointerToOption(err.Error_.GetStatus()),
		message: mo.PointerToOption(err.Error_.GetMessage()),
		reason:  mo.PointerToOption(err.Error_.GetReason()),
	}
}

func (e *censysClientUnauthorizedError) Error() string {
	var sb strings.Builder

	if e.code.IsPresent() {
		sb.WriteString(fmt.Sprintf("Code: %d\n", e.code.MustGet()))
	}
	if e.status.IsPresent() {
		sb.WriteString(fmt.Sprintf("Status: %s\n", e.status.MustGet()))
	}
	if e.message.IsPresent() {
		sb.WriteString(fmt.Sprintf("Message: %s\n", e.message.MustGet()))
	}
	if e.reason.IsPresent() {
		sb.WriteString(fmt.Sprintf("Reason: %s\n", e.reason.MustGet()))
	}

	return strings.TrimSpace(sb.String())
}

func (e *censysClientUnauthorizedError) Title() string {
	return "Unauthorized to Access Censys API"
}

func (e *censysClientUnauthorizedError) ShouldPrintUsage() bool {
	return false
}

func (e *censysClientUnauthorizedError) Status() string {
	if e.code.IsPresent() {
		statusText := http.StatusText(int(e.code.MustGet()))
		if statusText != "" {
			return statusText
		}
	}
	return "unknown"
}

func (e *censysClientUnauthorizedError) StatusCode() mo.Option[int64] {
	return e.code
}

type ClientGenericError interface {
	ClientError
}

type censysClientGenericError struct {
	message    string
	statusCode int
	body       string
}

var _ ClientGenericError = &censysClientGenericError{}

func NewCensysClientGenericError(err *sdkerrors.SDKError) ClientGenericError {
	return &censysClientGenericError{
		message:    err.Message,
		statusCode: err.StatusCode,
		body:       err.Body,
	}
}

func (e *censysClientGenericError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (status code: %d)\n", e.message, e.statusCode))
	bodyStr := ""
	var bodyMap map[string]any
	if err := json.Unmarshal([]byte(e.body), &bodyMap); err == nil {
		bodyBytes, marhsalErr := json.MarshalIndent(bodyMap, "", "  ")
		if marhsalErr == nil {
			bodyStr = string(bodyBytes)
		} else {
			bodyStr = fmt.Sprintf("%v", bodyMap)
		}
	} else {
		// if not json, truncate to 200 characters
		maxBodyLength := 200
		if len(e.body) > maxBodyLength {
			bodyStr = e.body[:maxBodyLength] + fmt.Sprintf("... (truncated %d bytes)", len(e.body)-maxBodyLength)
		} else {
			bodyStr = e.body
		}
	}
	sb.WriteString(bodyStr)
	return strings.TrimSpace(sb.String())
}

func (e *censysClientGenericError) Title() string {
	if e.statusCode == http.StatusTooManyRequests {
		return "Rate Limit Exceeded"
	}
	return "Error Returned from Censys API"
}

func (e *censysClientGenericError) ShouldPrintUsage() bool {
	return false
}

func (e *censysClientGenericError) Status() string {
	httpStatusText := http.StatusText(e.statusCode)
	if httpStatusText != "" {
		return httpStatusText
	}
	return "unknown"
}

func (e *censysClientGenericError) StatusCode() mo.Option[int64] {
	return mo.Some(int64(e.statusCode))
}

// CensysClientNotConfiguredError isn't really a client error, since
// it will be used before an API call is made.
type ClientNotConfiguredError interface {
	cenclierrors.CencliError
}

type censysClientNotConfiguredError struct{}

var _ ClientNotConfiguredError = &censysClientNotConfiguredError{}

func NewCensysClientNotConfiguredError() ClientNotConfiguredError {
	return &censysClientNotConfiguredError{}
}

func (e *censysClientNotConfiguredError) Error() string { return "The API client is not configured." }

func (e *censysClientNotConfiguredError) Title() string {
	return "Censys Client Not Configured"
}

func (e *censysClientNotConfiguredError) ShouldPrintUsage() bool {
	return false
}
