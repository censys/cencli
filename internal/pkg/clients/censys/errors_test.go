package censys

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface conformance checks
var (
	_ ClientError = (*clientErrorAdapter)(nil)
	_ ClientError = (*censysClientUnauthorizedError)(nil)
	_ ClientError = (*censysClientGenericError)(nil)
	_ ClientError = (*censysClientError)(nil)
)

func TestParseSDKError_Comprehensive(t *testing.T) {
	tests := []struct {
		name           string
		inputError     error
		expectedType   string // Type assertion check
		expectedStatus string
		expectedCode   mo.Option[int64]
		expectedOutput string // Exact Error() string match
		expectedTitle  string // Expected Title() string
	}{
		{
			name: "ErrorModel with all fields",
			inputError: &sdkerrors.ErrorModel{
				Detail:   strPtr("Request validation failed"),
				Title:    strPtr("Bad Request"),
				Status:   int64Ptr(400),
				Type:     strPtr("validation_error"),
				Instance: strPtr("/api/v2/hosts/search"),
				Errors: []components.ErrorDetail{
					{
						Location: strPtr("query.field"),
						Message:  strPtr("Field is required"),
						Value:    "invalid_value",
					},
				},
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Bad Request",
			expectedCode:   mo.Some(int64(400)),
			expectedOutput: "Request validation failed (400 Bad Request)\n" +
				"  - query.field: Field is required (got \"invalid_value\")",
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name: "ErrorModel with minimal fields",
			inputError: &sdkerrors.ErrorModel{
				Detail: strPtr("Resource not found"),
				Status: int64Ptr(404),
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Not Found",
			expectedCode:   mo.Some(int64(404)),
			expectedOutput: "Resource not found (404 Not Found)",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "ErrorModel with only a title and a nonstandard status",
			inputError: &sdkerrors.ErrorModel{
				Title:  strPtr("Something odd"),
				Status: int64Ptr(599),
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "unknown",
			expectedCode:   mo.Some(int64(599)),
			expectedOutput: "Something odd (599)",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "ErrorModel with a bare field error",
			inputError: &sdkerrors.ErrorModel{
				Detail: strPtr("Unprocessable Entity"),
				Status: int64Ptr(422),
				Errors: []components.ErrorDetail{{Message: strPtr("hostname must contain a period")}},
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Unprocessable Entity",
			expectedCode:   mo.Some(int64(422)),
			expectedOutput: "Unprocessable Entity (422 Unprocessable Entity)\n" +
				"  - hostname must contain a period",
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name: "ErrorModel whose title adds to the detail",
			inputError: &sdkerrors.ErrorModel{
				Title:  strPtr("Permission denied"),
				Detail: strPtr("You do not have access to this tag"),
				Status: int64Ptr(403),
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Forbidden",
			expectedCode:   mo.Some(int64(403)),
			expectedOutput: "Permission denied: You do not have access to this tag (403 Forbidden)",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "ErrorModel with an empty detail falls back to the title",
			inputError: &sdkerrors.ErrorModel{
				Title:  strPtr("Something odd"),
				Detail: strPtr(""),
				Status: int64Ptr(404),
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Not Found",
			expectedCode:   mo.Some(int64(404)),
			expectedOutput: "Something odd (404 Not Found)",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "ErrorModel field values are JSON-encoded",
			inputError: &sdkerrors.ErrorModel{
				Detail: strPtr("Request validation failed"),
				Status: int64Ptr(422),
				Errors: []components.ErrorDetail{
					{Location: strPtr("body.query"), Message: strPtr("invalid query"), Value: "host.name: a\nand b"},
					{Location: strPtr("body.target"), Message: strPtr("unknown field"), Value: map[string]any{"port": 443}},
				},
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Unprocessable Entity",
			expectedCode:   mo.Some(int64(422)),
			expectedOutput: "Request validation failed (422 Unprocessable Entity)\n" +
				"  - body.query: invalid query (got \"host.name: a\\nand b\")\n" +
				"  - body.target: unknown field (got {\"port\":443})",
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name: "ErrorModel control characters in server strings are escaped",
			inputError: &sdkerrors.ErrorModel{
				Title:  strPtr("Bad\x1b[31m title"),
				Detail: strPtr("invalid query:\nline twö"),
				Status: int64Ptr(422),
				Errors: []components.ErrorDetail{{Location: strPtr("body.\tquery"), Message: strPtr("bad\rinput")}},
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Unprocessable Entity",
			expectedCode:   mo.Some(int64(422)),
			expectedOutput: "Bad\\x1b[31m title: invalid query:\\nline twö (422 Unprocessable Entity)\n" +
				"  - body.\\tquery: bad\\rinput",
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name: "ErrorModel format characters and C1 controls are escaped, including in values",
			inputError: &sdkerrors.ErrorModel{
				Detail: strPtr("name \u202eevil\u200b"),
				Status: int64Ptr(422),
				Errors: []components.ErrorDetail{{Location: strPtr("body.name"), Message: strPtr("invalid"), Value: "a\u0085b\u202ec"}},
			},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "Unprocessable Entity",
			expectedCode:   mo.Some(int64(422)),
			expectedOutput: `name \u202eevil\u200b (422 Unprocessable Entity)` + "\n" +
				`  - body.name: invalid (got "a\u0085b\u202ec")`,
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name:           "ErrorModel with no fields",
			inputError:     &sdkerrors.ErrorModel{},
			expectedType:   "CensysClientStructuredError",
			expectedStatus: "unknown",
			expectedCode:   mo.None[int64](),
			expectedOutput: "the API returned an error without details",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "AuthenticationError",
			inputError: &sdkerrors.AuthenticationError{
				Error_: &components.AuthenticationErrorDetail{
					Code:    int64Ptr(401),
					Status:  strPtr("Unauthorized"),
					Message: strPtr("Invalid API key"),
					Reason:  strPtr("api_key_invalid"),
				},
			},
			expectedType:   "CensysClientUnauthorizedError",
			expectedStatus: "Unauthorized",
			expectedCode:   mo.Some(int64(401)),
			expectedOutput: "Code: 401\n" +
				"Status: Unauthorized\n" +
				"Message: Invalid API key\n" +
				"Reason: api_key_invalid",
			expectedTitle: "Unauthorized to Access Censys API",
		},
		{
			name: "AuthenticationError with minimal fields",
			inputError: &sdkerrors.AuthenticationError{
				Error_: &components.AuthenticationErrorDetail{
					Code: int64Ptr(403),
				},
			},
			expectedType:   "CensysClientUnauthorizedError",
			expectedStatus: "Forbidden",
			expectedCode:   mo.Some(int64(403)),
			expectedOutput: "Code: 403",
			expectedTitle:  "Unauthorized to Access Censys API",
		},
		{
			name: "SDKError",
			inputError: &sdkerrors.SDKError{
				Message:    "Internal server error occurred",
				StatusCode: 500,
				Body:       `{"error": "database connection failed"}`,
			},
			expectedType:   "CensysClientGenericError",
			expectedStatus: "Internal Server Error",
			expectedCode:   mo.Some(int64(500)),
			expectedOutput: "Internal server error occurred (status code: 500)\n" +
				"{\n" +
				"  \"error\": \"database connection failed\"\n" +
				"}",
			expectedTitle: "Error Returned from Censys API",
		},
		{
			name: "SDKError with empty body",
			inputError: &sdkerrors.SDKError{
				Message:    "Gateway timeout",
				StatusCode: 504,
				Body:       "",
			},
			expectedType:   "CensysClientGenericError",
			expectedStatus: "Gateway Timeout",
			expectedCode:   mo.Some(int64(504)),
			expectedOutput: "Gateway timeout (status code: 504)",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name: "SDKError with unknown status code",
			inputError: &sdkerrors.SDKError{
				Message:    "Custom error",
				StatusCode: 999,
				Body:       "custom error body",
			},
			expectedType:   "CensysClientGenericError",
			expectedStatus: "unknown",
			expectedCode:   mo.Some(int64(999)),
			expectedOutput: "Custom error (status code: 999)\ncustom error body",
			expectedTitle:  "Error Returned from Censys API",
		},
		{
			name:           "Unknown error type",
			inputError:     errors.New("network connection failed"),
			expectedType:   "ClientErrorAdapter",
			expectedStatus: "unknown",
			expectedCode:   mo.None[int64](),
			expectedOutput: "network connection failed",
			expectedTitle:  "Unknown Error",
		},
		{
			name:           "Wrapped unknown error",
			inputError:     errors.New("wrapped: original error"),
			expectedType:   "ClientErrorAdapter",
			expectedStatus: "unknown",
			expectedCode:   mo.None[int64](),
			expectedOutput: "wrapped: original error",
			expectedTitle:  "Unknown Error",
		},
		{
			name:           "Context canceled",
			inputError:     context.Canceled,
			expectedType:   "ClientErrorAdapter",
			expectedStatus: "unknown",
			expectedCode:   mo.None[int64](),
			expectedOutput: "the operation's context was cancelled before it completed",
			expectedTitle:  "Interrupted",
		},
		{
			name:           "Context deadline exceeded",
			inputError:     context.DeadlineExceeded,
			expectedType:   "ClientErrorAdapter",
			expectedStatus: "unknown",
			expectedCode:   mo.None[int64](),
			expectedOutput: "the operation timed out before it could be completed",
			expectedTitle:  "Timeout",
		},
		{
			name: "SDKError with 429 status and large body",
			inputError: &sdkerrors.SDKError{
				Message:    "Rate limit exceeded",
				StatusCode: 429,
				Body:       strings.Repeat("a", 300), // 300 character body to trigger truncation
			},
			expectedType:   "CensysClientGenericError",
			expectedStatus: "Too Many Requests",
			expectedCode:   mo.Some(int64(429)),
			expectedOutput: "Rate limit exceeded (status code: 429)\n" +
				strings.Repeat("a", 200) + "... (truncated 100 bytes)",
			expectedTitle: "Rate Limit Exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewClientError(tt.inputError)
			require.NotNil(t, result, "parseSDKError should never return nil")

			// Test type assertion
			switch tt.expectedType {
			case "CensysClientStructuredError":
				_, ok := result.(ClientStructuredError)
				assert.True(t, ok, "Expected CensysClientStructuredError type")
			case "CensysClientUnauthorizedError":
				_, ok := result.(ClientUnauthorizedError)
				assert.True(t, ok, "Expected CensysClientUnauthorizedError type")
			case "CensysClientGenericError":
				_, ok := result.(ClientGenericError)
				assert.True(t, ok, "Expected CensysClientGenericError type")
			case "ClientErrorAdapter":
				_, ok := result.(*clientErrorAdapter)
				assert.True(t, ok, "Expected clientErrorAdapter type")
			default:
				t.Fatalf("Unknown expected type: %s", tt.expectedType)
			}

			// Test Status() method
			assert.Equal(t, tt.expectedStatus, result.Status(), "Status() should match expected")

			// Test StatusCode() method
			if tt.expectedCode.IsPresent() {
				assert.True(t, result.StatusCode().IsPresent(), "StatusCode should be present")
				assert.Equal(t, tt.expectedCode.MustGet(), result.StatusCode().MustGet(), "StatusCode value should match")
			} else {
				assert.False(t, result.StatusCode().IsPresent(), "StatusCode should not be present")
			}

			// Test Error() method - exact string match
			assert.Equal(t, tt.expectedOutput, result.Error(), "Error() output should match exactly")

			// Test that it implements CensysClientError interface
			var clientErr ClientError
			assert.True(t, errors.As(result, &clientErr), "Result should implement CensysClientError")

			// Test common interface methods
			if tt.expectedTitle != "" {
				assert.Equal(t, tt.expectedTitle, result.Title(), "Title() should match expected")
			} else {
				assert.NotEmpty(t, result.Title(), "Title() should not be empty")
			}
			assert.False(t, result.ShouldPrintUsage(), "ShouldPrintUsage() should be false for client errors")
		})
	}
}

// Helper functions for creating pointers
func int64Ptr(i int64) *int64 {
	return &i
}

func strPtr(s string) *string { return &s }

func TestIsFeatureNotEnabled(t *testing.T) {
	problem := func(status int64, detail string) error {
		return NewCensysClientStructuredError(&sdkerrors.ErrorModel{Status: int64Ptr(status), Detail: strPtr(detail)})
	}
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "feature flag 409", err: problem(409, "feature is not enabled"), expected: true},
		{name: "other 409", err: problem(409, "resource already exists")},
		{name: "same detail, other status", err: problem(403, "feature is not enabled")},
		{name: "generic error", err: errors.New("feature is not enabled")},
		{name: "nil", err: nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsFeatureNotEnabled(tc.err))
		})
	}
}
