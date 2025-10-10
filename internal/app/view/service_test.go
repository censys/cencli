package view

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func TestViewService_GetHosts(t *testing.T) {
	t.Run("success without orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")

		mockClient.EXPECT().GetHosts(gomock.Any(), mo.None[string](), []string{"8.8.8.8"}, mo.None[time.Time]()).Return(client.Result[[]components.Host]{
			Data: &[]components.Host{{IP: strPtr("8.8.8.8")}},
			Metadata: client.Metadata{
				Request: &http.Request{
					Method: "POST",
					URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
				},
				Response: &http.Response{
					StatusCode: 200,
				},
				Latency: 100 * time.Millisecond,
			},
		}, nil)

		svc := New(mockClient)
		res, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]())

		require.Nil(t, err)
		require.Len(t, res.Hosts, 1)
		require.Equal(t, "8.8.8.8", *res.Hosts[0].IP)
		assert.Equal(t, "POST", res.Meta.Method)
		assert.Equal(t, 200, res.Meta.Status)
	})

	t.Run("success with orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		now := time.Now()

		mockClient := mocks.NewMockClient(ctrl)
		hostID, _ := assets.NewHostID("192.168.1.1")
		orgID := identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))

		mockClient.EXPECT().GetHosts(gomock.Any(), mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"), []string{"192.168.1.1"}, mo.Some(now)).Return(client.Result[[]components.Host]{
			Data: &[]components.Host{{IP: strPtr("192.168.1.1")}},
			Metadata: client.Metadata{
				Request: &http.Request{
					Method: "POST",
					URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
				},
				Response: &http.Response{
					StatusCode: 200,
				},
				Latency: 100 * time.Millisecond,
			},
		}, nil)

		svc := New(mockClient)
		res, err := svc.GetHosts(context.Background(), mo.Some(orgID), []assets.HostID{hostID}, mo.Some(now))

		require.Nil(t, err)
		require.Len(t, res.Hosts, 1)
		require.Equal(t, "192.168.1.1", *res.Hosts[0].IP)
	})

	t.Run("client structured error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")

		detail := "Invalid host ID format"
		status := int64(400)
		structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
			Detail: &detail,
			Status: &status,
		})

		mockClient.EXPECT().GetHosts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
			client.Result[[]components.Host]{},
			structuredErr,
		)

		svc := New(mockClient)
		_, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]())

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "Invalid host ID format")
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("client generic error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")

		genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
			Message:    "Rate limit exceeded",
			StatusCode: 429,
			Body:       "Too many requests",
		})

		mockClient.EXPECT().GetHosts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
			client.Result[[]components.Host]{},
			genericErr,
		)

		svc := New(mockClient)
		_, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]())

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "Rate limit exceeded")
		assert.Contains(t, err.Error(), "429")
	})

	t.Run("client unknown error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")

		unknownErr := client.NewClientError(errors.New("network timeout"))

		mockClient.EXPECT().GetHosts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
			client.Result[[]components.Host]{},
			unknownErr,
		)

		svc := New(mockClient)
		_, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]())

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "network timeout")
	})
}

func TestViewService_GetCertificates(t *testing.T) {
	t.Run("success without orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		fingerprint := "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"
		certID, _ := assets.NewCertificateFingerprint(fingerprint)

		mockClient.EXPECT().GetCertificates(gomock.Any(), mo.None[string](), []string{fingerprint}).Return(
			client.Result[[]components.Certificate]{
				Data: &[]components.Certificate{{FingerprintSha256: strPtr(fingerprint)}},
				Metadata: client.Metadata{
					Request: &http.Request{
						Method: "POST",
						URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
					},
					Response: &http.Response{
						StatusCode: 200,
					},
					Latency: 150 * time.Millisecond,
				},
			}, nil)

		svc := New(mockClient)
		res, err := svc.GetCertificates(context.Background(), mo.None[identifiers.OrganizationID](), []assets.CertificateID{certID})

		require.Nil(t, err)
		require.Len(t, res.Certificates, 1)
		require.Equal(t, fingerprint, *res.Certificates[0].FingerprintSha256)
		assert.Equal(t, "POST", res.Meta.Method)
		assert.Equal(t, 200, res.Meta.Status)
	})

	t.Run("success with orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		fingerprint := "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"
		certID, _ := assets.NewCertificateFingerprint(fingerprint)
		orgID := identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))

		mockClient.EXPECT().GetCertificates(gomock.Any(), mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"), []string{fingerprint}).Return(
			client.Result[[]components.Certificate]{
				Data: &[]components.Certificate{{FingerprintSha256: strPtr(fingerprint)}},
				Metadata: client.Metadata{
					Request: &http.Request{
						Method: "POST",
						URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
					},
					Response: &http.Response{
						StatusCode: 200,
					},
					Latency: 150 * time.Millisecond,
				},
			}, nil)

		svc := New(mockClient)
		res, err := svc.GetCertificates(context.Background(), mo.Some(orgID), []assets.CertificateID{certID})

		require.Nil(t, err)
		require.Len(t, res.Certificates, 1)
	})

	t.Run("client error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		fingerprint := "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"
		certID, _ := assets.NewCertificateFingerprint(fingerprint)

		detail := "Certificate not found"
		status := int64(404)
		structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
			Detail: &detail,
			Status: &status,
		})

		mockClient.EXPECT().GetCertificates(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			client.Result[[]components.Certificate]{},
			structuredErr,
		)

		svc := New(mockClient)
		_, err := svc.GetCertificates(context.Background(), mo.None[identifiers.OrganizationID](), []assets.CertificateID{certID})

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "Certificate not found")
		assert.Contains(t, err.Error(), "404")
	})
}

func TestViewService_GetWebProperties(t *testing.T) {
	t.Run("success without orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		webPropID, _ := assets.NewWebPropertyID("example.com:443", assets.DefaultWebPropertyPort)

		mockClient.EXPECT().GetWebProperties(gomock.Any(), mo.None[string](), []string{"example.com:443"}, mo.None[time.Time]()).Return(
			client.Result[[]components.Webproperty]{
				Data: &[]components.Webproperty{{
					Hostname: strPtr("example.com"),
					Port:     intPtr(443),
				}},
				Metadata: client.Metadata{
					Request: &http.Request{
						Method: "POST",
						URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
					},
					Response: &http.Response{
						StatusCode: 200,
					},
					Latency: 200 * time.Millisecond,
				},
			}, nil)

		svc := New(mockClient)
		res, err := svc.GetWebProperties(context.Background(), mo.None[identifiers.OrganizationID](), []assets.WebPropertyID{webPropID}, mo.None[time.Time]())

		require.Nil(t, err)
		require.Len(t, res.WebProperties, 1)
		require.Equal(t, "example.com", *res.WebProperties[0].Hostname)
		require.Equal(t, 443, *res.WebProperties[0].Port)
		assert.Equal(t, "POST", res.Meta.Method)
		assert.Equal(t, 200, res.Meta.Status)
	})

	t.Run("success with orgID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		now := time.Now()

		webPropID, _ := assets.NewWebPropertyID("test.com:80", assets.DefaultWebPropertyPort)
		orgID := identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))

		mockClient.EXPECT().GetWebProperties(gomock.Any(), mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"), []string{"test.com:80"}, mo.Some(now)).Return(
			client.Result[[]components.Webproperty]{
				Data: &[]components.Webproperty{{
					Hostname: strPtr("test.com"),
					Port:     intPtr(80),
				}},
				Metadata: client.Metadata{
					Request: &http.Request{
						Method: "POST",
						URL:    &url.URL{Scheme: "https", Host: "api.censys.io"},
					},
					Response: &http.Response{
						StatusCode: 200,
					},
					Latency: 200 * time.Millisecond,
				},
			}, nil)

		svc := New(mockClient)
		res, err := svc.GetWebProperties(context.Background(), mo.Some(orgID), []assets.WebPropertyID{webPropID}, mo.Some(now))

		require.Nil(t, err)
		require.Len(t, res.WebProperties, 1)
		require.Equal(t, "test.com", *res.WebProperties[0].Hostname)
		require.Equal(t, 80, *res.WebProperties[0].Port)
	})

	t.Run("client error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		webPropID, _ := assets.NewWebPropertyID("example.com:443", assets.DefaultWebPropertyPort)

		genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
			Message:    "Service unavailable",
			StatusCode: 503,
			Body:       "Service temporarily unavailable",
		})

		mockClient.EXPECT().GetWebProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
			client.Result[[]components.Webproperty]{},
			genericErr,
		)

		svc := New(mockClient)
		_, err := svc.GetWebProperties(context.Background(), mo.None[identifiers.OrganizationID](), []assets.WebPropertyID{webPropID}, mo.None[time.Time]())

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "Service unavailable")
		assert.Contains(t, err.Error(), "503")
	})
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// TestViewService_GetHosts_Batching tests the batching functionality for hosts
func TestViewService_GetHosts_Batching(t *testing.T) {
	t.Run("batches hosts when exceeding API limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 150 host IDs (exceeds the 100 limit)
		var hostIDs []assets.HostID
		for i := 1; i <= 150; i++ {
			hostID, _ := assets.NewHostID(fmt.Sprintf("10.0.0.%d", i))
			hostIDs = append(hostIDs, hostID)
		}

		// Expect two batches: 100 + 50
		// First batch
		mockClient.EXPECT().GetHosts(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(100),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Host]{
			Data: &[]components.Host{{IP: strPtr("10.0.0.1")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  100 * time.Millisecond,
			},
		}, nil)

		// Second batch
		mockClient.EXPECT().GetHosts(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(50),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Host]{
			Data: &[]components.Host{{IP: strPtr("10.0.0.101")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  100 * time.Millisecond,
			},
		}, nil)

		svc := New(mockClient)
		res, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs, mo.None[time.Time]())

		require.Nil(t, err)
		require.Nil(t, res.PartialError)
		require.Len(t, res.Hosts, 2) // One from each batch
		assert.Equal(t, uint64(2), res.Meta.PageCount)
	})

	t.Run("error on second batch returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 150 host IDs
		var hostIDs []assets.HostID
		for i := 1; i <= 150; i++ {
			hostID, _ := assets.NewHostID(fmt.Sprintf("10.0.0.%d", i))
			hostIDs = append(hostIDs, hostID)
		}

		// First batch succeeds
		mockClient.EXPECT().GetHosts(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(100),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Host]{
			Data: &[]components.Host{{IP: strPtr("10.0.0.1")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  100 * time.Millisecond,
			},
		}, nil)

		// Second batch fails
		mockClient.EXPECT().GetHosts(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(50),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Host]{}, client.NewClientError(errors.New("network error")))

		svc := New(mockClient)
		res, err := svc.GetHosts(context.Background(), mo.None[identifiers.OrganizationID](), hostIDs, mo.None[time.Time]())

		require.Nil(t, err)
		require.NotNil(t, res.PartialError)
		require.Contains(t, res.PartialError.Error(), "network error")
		require.Len(t, res.Hosts, 1) // Only first batch
		assert.Equal(t, uint64(1), res.Meta.PageCount)
	})

	t.Run("context cancelled between batches returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 150 host IDs
		var hostIDs []assets.HostID
		for i := 1; i <= 150; i++ {
			hostID, _ := assets.NewHostID(fmt.Sprintf("10.0.0.%d", i))
			hostIDs = append(hostIDs, hostID)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// First batch succeeds, then cancel
		mockClient.EXPECT().GetHosts(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(100),
			mo.None[time.Time](),
		).DoAndReturn(func(ctx context.Context, orgID mo.Option[string], hostIDs []string, atTime mo.Option[time.Time]) (client.Result[[]components.Host], client.ClientError) {
			cancel() // Cancel after first batch
			return client.Result[[]components.Host]{
				Data: &[]components.Host{{IP: strPtr("10.0.0.1")}},
				Metadata: client.Metadata{
					Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
					Response: &http.Response{StatusCode: 200},
					Latency:  100 * time.Millisecond,
				},
			}, nil
		})

		svc := New(mockClient)
		res, err := svc.GetHosts(ctx, mo.None[identifiers.OrganizationID](), hostIDs, mo.None[time.Time]())

		require.Nil(t, err)
		require.NotNil(t, res.PartialError)
		require.ErrorIs(t, res.PartialError, context.Canceled)
		require.Len(t, res.Hosts, 1) // Only first batch
	})
}

// TestViewService_GetCertificates_Batching tests the batching functionality for certificates
func TestViewService_GetCertificates_Batching(t *testing.T) {
	t.Run("batches certificates when exceeding API limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 1500 certificate IDs (exceeds the 1000 limit)
		var certIDs []assets.CertificateID
		for i := 1; i <= 1500; i++ {
			certID, _ := assets.NewCertificateFingerprint(fmt.Sprintf("%064d", i))
			certIDs = append(certIDs, certID)
		}

		// Expect two batches: 1000 + 500
		mockClient.EXPECT().GetCertificates(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(1000),
		).Return(client.Result[[]components.Certificate]{
			Data: &[]components.Certificate{{FingerprintSha256: strPtr("cert1")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  150 * time.Millisecond,
			},
		}, nil)

		mockClient.EXPECT().GetCertificates(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(500),
		).Return(client.Result[[]components.Certificate]{
			Data: &[]components.Certificate{{FingerprintSha256: strPtr("cert2")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  150 * time.Millisecond,
			},
		}, nil)

		svc := New(mockClient)
		res, err := svc.GetCertificates(context.Background(), mo.None[identifiers.OrganizationID](), certIDs)

		require.Nil(t, err)
		require.Nil(t, res.PartialError)
		require.Len(t, res.Certificates, 2)
		assert.Equal(t, uint64(2), res.Meta.PageCount)
	})

	t.Run("error on second batch returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 1500 certificate IDs
		var certIDs []assets.CertificateID
		for i := 1; i <= 1500; i++ {
			certID, _ := assets.NewCertificateFingerprint(fmt.Sprintf("%064d", i))
			certIDs = append(certIDs, certID)
		}

		// First batch succeeds
		mockClient.EXPECT().GetCertificates(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(1000),
		).Return(client.Result[[]components.Certificate]{
			Data: &[]components.Certificate{{FingerprintSha256: strPtr("cert1")}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  150 * time.Millisecond,
			},
		}, nil)

		// Second batch fails
		mockClient.EXPECT().GetCertificates(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(500),
		).Return(client.Result[[]components.Certificate]{}, client.NewClientError(errors.New("rate limit exceeded")))

		svc := New(mockClient)
		res, err := svc.GetCertificates(context.Background(), mo.None[identifiers.OrganizationID](), certIDs)

		require.Nil(t, err)
		require.NotNil(t, res.PartialError)
		require.Contains(t, res.PartialError.Error(), "rate limit exceeded")
		require.Len(t, res.Certificates, 1)
		assert.Equal(t, uint64(1), res.Meta.PageCount)
	})
}

// TestViewService_GetWebProperties_Batching tests the batching functionality for web properties
func TestViewService_GetWebProperties_Batching(t *testing.T) {
	t.Run("batches web properties when exceeding API limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 150 web property IDs (exceeds the 100 limit)
		var webPropIDs []assets.WebPropertyID
		for i := 1; i <= 150; i++ {
			webPropID, _ := assets.NewWebPropertyID(fmt.Sprintf("example%d.com:443", i), assets.DefaultWebPropertyPort)
			webPropIDs = append(webPropIDs, webPropID)
		}

		// Expect two batches: 100 + 50
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(100),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Webproperty]{
			Data: &[]components.Webproperty{{Hostname: strPtr("example1.com"), Port: intPtr(443)}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  200 * time.Millisecond,
			},
		}, nil)

		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(50),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Webproperty]{
			Data: &[]components.Webproperty{{Hostname: strPtr("example101.com"), Port: intPtr(443)}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  200 * time.Millisecond,
			},
		}, nil)

		svc := New(mockClient)
		res, err := svc.GetWebProperties(context.Background(), mo.None[identifiers.OrganizationID](), webPropIDs, mo.None[time.Time]())

		require.Nil(t, err)
		require.Nil(t, res.PartialError)
		require.Len(t, res.WebProperties, 2)
		assert.Equal(t, uint64(2), res.Meta.PageCount)
	})

	t.Run("error on second batch returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)

		// Create 150 web property IDs
		var webPropIDs []assets.WebPropertyID
		for i := 1; i <= 150; i++ {
			webPropID, _ := assets.NewWebPropertyID(fmt.Sprintf("example%d.com:443", i), assets.DefaultWebPropertyPort)
			webPropIDs = append(webPropIDs, webPropID)
		}

		// First batch succeeds
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(100),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Webproperty]{
			Data: &[]components.Webproperty{{Hostname: strPtr("example1.com"), Port: intPtr(443)}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  200 * time.Millisecond,
			},
		}, nil)

		// Second batch fails
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			gomock.Len(50),
			mo.None[time.Time](),
		).Return(client.Result[[]components.Webproperty]{}, client.NewClientError(errors.New("service unavailable")))

		svc := New(mockClient)
		res, err := svc.GetWebProperties(context.Background(), mo.None[identifiers.OrganizationID](), webPropIDs, mo.None[time.Time]())

		require.Nil(t, err)
		require.NotNil(t, res.PartialError)
		require.Contains(t, res.PartialError.Error(), "service unavailable")
		require.Len(t, res.WebProperties, 1)
		assert.Equal(t, uint64(1), res.Meta.PageCount)
	})
}
