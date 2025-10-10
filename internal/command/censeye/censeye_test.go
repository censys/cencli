package censeye

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	censeyemocks "github.com/censys/cencli/gen/app/censeye/mocks"
	viewmocks "github.com/censys/cencli/gen/app/view/mocks"
	"github.com/censys/cencli/internal/app/censeye"
	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func TestCenseyeCommand(t *testing.T) {
	testCases := []struct {
		name       string
		viewSvc    func(ctrl *gomock.Controller) view.Service
		censeyeSvc func(ctrl *gomock.Controller) censeye.Service
		setup      func(t *testing.T, tempDir string, args *[]string)
		stdin      string
		args       []string
		assert     func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - default rarity bounds",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
						{
							Count:       100,
							Query:       `services.service_name="HTTP"`,
							Interesting: false,
							SearchURL:   "https://platform.censys.io/search?q=services.service_name%3D%22HTTP%22",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 8.8.8.8")
				require.Contains(t, stdout, "10")
				require.Contains(t, stdout, "100")
				require.Contains(t, stdout, "services.port=80")
				require.Contains(t, stdout, `services.service_name="HTTP"`)
				require.Contains(t, stdout, "Found 1 interesting of 2 within [2,100]")
				require.Contains(t, stdout, "Pivots (interesting queries)")
			},
		},
		{
			name: "success - custom rarity bounds",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("1.1.1.1")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("1.1.1.1"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       5,
							Query:       `services.port=443`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D443",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(5), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"1.1.1.1", "--rarity-min", "5", "--rarity-max", "100"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 1.1.1.1")
				require.Contains(t, stdout, "Found 1 interesting of 1 within [5,100]")
			},
		},
		{
			name: "success - raw output",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--raw"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Verify valid JSON
				var entries []censeye.ReportEntry
				jsonErr := json.Unmarshal([]byte(stdout), &entries)
				require.NoError(t, jsonErr)
				require.Len(t, entries, 1)
				require.Equal(t, int64(10), entries[0].Count)
				require.Equal(t, `services.port=80`, entries[0].Query)
				require.True(t, entries[0].Interesting)
				// search_url should be stripped in raw output by default
				require.Empty(t, entries[0].SearchURL)
			},
		},
		{
			name: "success - raw output with url",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--raw", "--include-url"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Verify valid JSON
				var entries []censeye.ReportEntry
				jsonErr := json.Unmarshal([]byte(stdout), &entries)
				require.NoError(t, jsonErr)
				require.Len(t, entries, 1)
				require.Equal(t, int64(10), entries[0].Count)
				require.Equal(t, `services.port=80`, entries[0].Query)
				require.True(t, entries[0].Interesting)
				// search_url should be present with --include-url
				require.Equal(t, "https://platform.censys.io/search?q=services.port%3D80", entries[0].SearchURL)
			},
		},
		{
			name: "success - no interesting results",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       100,
							Query:       `services.port=80`,
							Interesting: false,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 8.8.8.8")
				require.Contains(t, stdout, "Found 0 interesting of 1 within [2,100]")
				require.NotContains(t, stdout, "Pivots (interesting queries)")
			},
		},
		{
			name: "success - empty results",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 8.8.8.8")
				require.Contains(t, stdout, "No rules found")
				require.Contains(t, stdout, "Found 0 interesting of 0 within [2,100]")
			},
		},
		{
			name: "success - with org-id",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				uuidVal, _ := uuid.Parse("a0000000-0000-0000-0000-000000000000")
				orgID := identifiers.NewOrganizationID(uuidVal)
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.Some(orgID), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				uuidVal, _ := uuid.Parse("a0000000-0000-0000-0000-000000000000")
				orgID := identifiers.NewOrganizationID(uuidVal)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.Some(orgID), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--org-id", "a0000000-0000-0000-0000-000000000000"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 8.8.8.8")
			},
		},
		{
			name: "error - host not found",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr HostNotFoundError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "host 8.8.8.8 not found")
			},
		},
		{
			name: "error - view service error",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "internal server error",
					StatusCode: 500,
					Body:       "error body",
				})
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(view.HostsResult{}, genericErr)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr client.ClientGenericError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "internal server error")
			},
		},
		{
			name: "error - censeye service error",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				unknownErr := client.NewClientError(errors.New("investigation failed"))
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(censeye.InvestigateHostResult{}, unknownErr)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr client.ClientError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "investigation failed")
			},
		},
		{
			name: "error - invalid rarity-min greater than rarity-max",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8", "--rarity-min", "100", "--rarity-max", "50"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr InvalidRarityFlagError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "rarity-min")
				require.Contains(t, err.Error(), "must be less than or equal to rarity-max")
			},
		},
		{
			name: "error - invalid rarity-min zero",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8", "--rarity-min", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr flags.IntegerFlagInvalidValueError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "rarity-min")
			},
		},
		{
			name: "error - invalid rarity-max zero",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8", "--rarity-max", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr flags.IntegerFlagInvalidValueError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "rarity-max")
			},
		},
		{
			name: "error - bad org-id",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8", "--org-id", "not-a-uuid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr flags.InvalidUUIDFlagError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - no argument",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				// Error message changed since we now support input-file flag
				var noAssetsErr assets.NoAssetsError
				require.ErrorAs(t, err, &noAssetsErr)
			},
		},
		{
			name: "error - invalid asset ID",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"8.8.8.8,1.1.1.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Equal(t, assets.NewTooManyAssetsError(2, 1), err)
			},
		},
		{
			name: "error - certificate not supported",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr ErrorAssetTypeNotSupportedError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "certificate")
				require.Contains(t, err.Error(), "not supported")
			},
		},
		{
			name: "error - web property not supported",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"platform.censys.io:443"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr ErrorAssetTypeNotSupportedError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "webproperty")
				require.Contains(t, err.Error(), "not supported")
			},
		},
		{
			name: "help message",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"--help"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Usage:")
				require.Contains(t, stdout, "censeye <asset>")
				require.Contains(t, stdout, "rarity-min")
				require.Contains(t, stdout, "rarity-max")
				require.Contains(t, stdout, "raw")
				require.Contains(t, stdout, "include-url")
			},
		},

		// Output format tests
		{
			name: "success - raw flag outputs JSON",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"--raw", "8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain JSON output
				var entries []censeye.ReportEntry
				require.NoError(t, json.Unmarshal([]byte(stdout), &entries))
				require.Len(t, entries, 1)
				require.Equal(t, int64(10), entries[0].Count)
				require.Equal(t, "services.port=80", entries[0].Query)
				require.True(t, entries[0].Interesting)
				// Should NOT contain table format
				require.NotContains(t, stdout, "CensEye Results")
			},
		},
		{
			name: "success - raw flag with include-url",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"--raw", "--include-url", "8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain JSON output with search_url
				var entries []censeye.ReportEntry
				require.NoError(t, json.Unmarshal([]byte(stdout), &entries))
				require.Len(t, entries, 1)
				require.NotEmpty(t, entries[0].SearchURL)
				require.Contains(t, entries[0].SearchURL, "platform.censys.io")
			},
		},
		{
			name: "success - default outputs raw table",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{
							Count:       10,
							Query:       `services.port=80`,
							Interesting: true,
							SearchURL:   "https://platform.censys.io/search?q=services.port%3D80",
						},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Should contain table output (not JSON)
				require.Contains(t, stdout, "CensEye Results for 8.8.8.8")
				require.Contains(t, stdout, "services.port=80")
				require.Contains(t, stdout, "10")
				require.Contains(t, stdout, "Found 1 interesting of 1 within [2,100]")
				// Should NOT be JSON format
				require.NotContains(t, stdout, `"count"`)
				require.NotContains(t, stdout, `"query"`)
			},
		},

		// Flag conflict tests
		{
			name: "error - raw and interactive flags together",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			args: []string{"--raw", "--interactive", "8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Equal(t, flags.NewConflictingFlagsError("raw", "interactive"), err)
			},
		},
		{
			name: "success - raw flag with input-file short form",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("8.8.8.8"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{Count: 10, Query: `test=query`, Interesting: true},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			setup: func(t *testing.T, tempDir string, args *[]string) {
				require.NoError(t, os.WriteFile(tempDir+"/test.txt", []byte("8.8.8.8\n"), 0o644))
				(*args)[len(*args)-1] = tempDir + "/test.txt"
			},
			args: []string{"-r", "-i", "test.txt"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				var entries []censeye.ReportEntry
				require.NoError(t, json.Unmarshal([]byte(stdout), &entries))
			},
		},

		// Asset input tests
		{
			name: "success - read asset from file",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("10.0.0.1")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("10.0.0.1"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{Count: 5, Query: `test=query`, Interesting: true},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			setup: func(t *testing.T, tempDir string, args *[]string) {
				require.NoError(t, os.WriteFile(tempDir+"/hosts.txt", []byte("10.0.0.1\n"), 0o644))
				(*args)[len(*args)-1] = tempDir + "/hosts.txt"
			},
			args: []string{"--input-file", "hosts.txt"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 10.0.0.1")
			},
		},
		{
			name: "success - read asset from stdin",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("10.0.0.2")
				host := &assets.Host{Host: components.Host{
					IP: strPtr("10.0.0.2"),
				}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				ms := censeyemocks.NewMockCenseyeService(ctrl)
				result := censeye.InvestigateHostResult{
					Entries: []censeye.ReportEntry{
						{Count: 5, Query: `test=query`, Interesting: true},
					},
				}
				ms.EXPECT().InvestigateHost(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), uint64(2), uint64(100)).Return(result, nil)
				return ms
			},
			stdin: "10.0.0.2\n",
			args:  []string{"--input-file", "-"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "CensEye Results for 10.0.0.2")
			},
		},
		{
			name: "error - no assets provided",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			setup: func(t *testing.T, tempDir string, args *[]string) {
				require.NoError(t, os.WriteFile(tempDir+"/empty.txt", []byte(""), 0o644))
				(*args)[len(*args)-1] = tempDir + "/empty.txt"
			},
			args: []string{"--input-file", "empty.txt"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var noAssetsErr assets.NoAssetsError
				require.ErrorAs(t, err, &noAssetsErr)
			},
		},
		{
			name: "error - too many assets provided",
			viewSvc: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			censeyeSvc: func(ctrl *gomock.Controller) censeye.Service {
				return censeyemocks.NewMockCenseyeService(ctrl)
			},
			setup: func(t *testing.T, tempDir string, args *[]string) {
				require.NoError(t, os.WriteFile(tempDir+"/multiple.txt", []byte("10.0.0.1\n10.0.0.2\n"), 0o644))
				(*args)[len(*args)-1] = tempDir + "/multiple.txt"
			},
			args: []string{"--input-file", "multiple.txt"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var tooManyErr assets.TooManyAssetsError
				require.ErrorAs(t, err, &tooManyErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			if tc.setup != nil {
				tc.setup(t, tempDir, &tc.args)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			viewSvc := tc.viewSvc(ctrl)
			censeyeSvc := tc.censeyeSvc(ctrl)
			cmdContext := command.NewCommandContext(cfg, nil, command.WithViewService(viewSvc), command.WithCenseyeService(censeyeSvc))
			rootCmd, err := command.RootCommandToCobra(NewCenseyeCommand(cmdContext))
			require.NoError(t, err)

			if tc.stdin != "" {
				rootCmd.SetIn(strings.NewReader(tc.stdin))
			}

			rootCmd.SetArgs(tc.args)

			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func strPtr(s string) *string { return &s }
