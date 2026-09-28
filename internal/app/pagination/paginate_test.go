package pagination

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/censys/censys-sdk-go/models/sdkerrors"

	"github.com/censys/cencli/internal/app/streaming"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

type testPage struct {
	items []int
	next  string
	total int64
}

func extractTestPage(p *testPage) PageData[int] {
	return PageData[int]{Items: p.items, TotalSize: p.total, NextPageToken: p.next}
}

// fetchFrom serves pages keyed by page token ("" is the first page). A key in
// failOn returns an API error instead of a page.
func fetchFrom(pages map[string]testPage, failOn ...string) func(mo.Option[string]) (client.Result[testPage], client.ClientError) {
	return func(token mo.Option[string]) (client.Result[testPage], client.ClientError) {
		key := token.OrElse("")
		for _, f := range failOn {
			if f == key {
				detail, status := "boom", int64(500)
				return client.Result[testPage]{}, client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
			}
		}
		p := pages[key]
		return client.Result[testPage]{Data: &p}, nil
	}
}

var threePages = map[string]testPage{
	"":   {items: []int{1, 2}, next: "p2", total: 5},
	"p2": {items: []int{3, 4}, next: "p3", total: 5},
	"p3": {items: []int{5}, next: "", total: 5},
}

func TestPaginate(t *testing.T) {
	testCases := []struct {
		name        string
		maxPages    mo.Option[uint64]
		pages       map[string]testPage
		failOn      []string
		wantItems   []int
		wantTotal   int64
		wantErr     bool
		wantPartial bool
	}{
		{name: "success - all pages", maxPages: mo.None[uint64](), pages: threePages, wantItems: []int{1, 2, 3, 4, 5}, wantTotal: 5},
		{name: "success - stops at max pages", maxPages: mo.Some[uint64](2), pages: threePages, wantItems: []int{1, 2, 3, 4}, wantTotal: 5},
		{name: "error - first page error is a hard error", maxPages: mo.None[uint64](), pages: threePages, failOn: []string{""}, wantErr: true},
		{name: "error - later page error is partial", maxPages: mo.None[uint64](), pages: threePages, failOn: []string{"p2"}, wantItems: []int{1, 2}, wantTotal: 5, wantPartial: true},
		{
			name:     "success - stops when the server repeats a token",
			maxPages: mo.None[uint64](),
			pages: map[string]testPage{
				"":   {items: []int{1}, next: "p2", total: 9},
				"p2": {items: []int{2}, next: "p2", total: 9},
			},
			wantItems: []int{1, 2},
			wantTotal: 9,
		},
		{name: "success - empty first page", maxPages: mo.None[uint64](), pages: map[string]testPage{"": {}}, wantItems: nil, wantTotal: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Paginate(context.Background(), tc.maxPages, "items", fetchFrom(tc.pages, tc.failOn...), extractTestPage)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantItems, res.Items)
			require.Equal(t, tc.wantTotal, res.TotalSize)
			if tc.wantPartial {
				require.Error(t, res.PartialError)
			} else {
				require.Nil(t, res.PartialError)
			}
		})
	}
}

func TestPaginate_Streaming(t *testing.T) {
	emitter, items := streaming.NewChannelEmitter(64)
	ctx := streaming.WithEmitter(context.Background(), emitter)

	res, err := Paginate(ctx, mo.None[uint64](), "items", fetchFrom(threePages), extractTestPage)
	require.NoError(t, err)
	require.Empty(t, res.Items, "streaming mode emits items instead of collecting them")
	require.Equal(t, int64(5), res.TotalSize)

	emitter.Close(nil)
	count := 0
	for item := range items {
		// Close sends a final Done item; it is not a record.
		if item.Done {
			break
		}
		count++
	}
	require.Equal(t, 5, count)
}
