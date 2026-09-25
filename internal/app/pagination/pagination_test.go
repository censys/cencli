package pagination

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

type testPage struct {
	items []string
	next  string
}

func okMeta() client.Metadata {
	return client.Metadata{
		Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
		Response: &http.Response{StatusCode: 200},
		Latency:  100 * time.Millisecond,
	}
}

func extract(p *testPage) Page[string] {
	return Page[string]{Items: p.items, NextPageToken: p.next}
}

// fetcher serves pages in order and records the tokens it was given.
func fetcher(pages []testPage, errAt int, tokens *[]mo.Option[string]) func(mo.Option[string]) (client.Result[testPage], client.ClientError) {
	i := 0
	return func(tok mo.Option[string]) (client.Result[testPage], client.ClientError) {
		*tokens = append(*tokens, tok)
		if i == errAt {
			i++
			return client.Result[testPage]{}, client.NewClientError(errors.New("boom"))
		}
		p := pages[i]
		i++
		return client.Result[testPage]{Metadata: okMeta(), Data: &p}, nil
	}
}

func TestPaginate(t *testing.T) {
	testCases := []struct {
		name        string
		pages       []testPage
		errAt       int
		maxPages    mo.Option[uint64]
		wantItems   []string
		wantCalls   int
		wantErr     bool
		wantPartial bool
		wantHasMore bool
	}{
		{
			name:      "single page without next token",
			pages:     []testPage{{items: []string{"a", "b"}}},
			errAt:     -1,
			maxPages:  mo.Some[uint64](5),
			wantItems: []string{"a", "b"},
			wantCalls: 1,
		},
		{
			name:        "stops at max pages",
			pages:       []testPage{{items: []string{"a"}, next: "t1"}, {items: []string{"b"}, next: "t2"}},
			errAt:       -1,
			maxPages:    mo.Some[uint64](1),
			wantItems:   []string{"a"},
			wantCalls:   1,
			wantHasMore: true,
		},
		{
			name:      "all pages until next token is empty",
			pages:     []testPage{{items: []string{"a"}, next: "t1"}, {items: []string{"b"}}},
			errAt:     -1,
			maxPages:  mo.None[uint64](),
			wantItems: []string{"a", "b"},
			wantCalls: 2,
		},
		{
			name:      "stops when the server echoes the same token",
			pages:     []testPage{{items: []string{"a"}, next: "t1"}, {items: []string{"b"}, next: "t1"}, {items: []string{"c"}}},
			errAt:     -1,
			maxPages:  mo.None[uint64](),
			wantItems: []string{"a", "b"},
			wantCalls: 2,
		},
		{
			name:      "error on first page is a hard error",
			pages:     []testPage{},
			errAt:     0,
			maxPages:  mo.None[uint64](),
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name:        "error on second page is a partial error",
			pages:       []testPage{{items: []string{"a"}, next: "t1"}},
			errAt:       1,
			maxPages:    mo.None[uint64](),
			wantItems:   []string{"a"},
			wantCalls:   2,
			wantPartial: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var tokens []mo.Option[string]
			res, err := Paginate(context.Background(), tc.maxPages, "things", fetcher(tc.pages, tc.errAt, &tokens), extract)
			require.Len(t, tokens, tc.wantCalls)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantItems, res.Items)
			require.Equal(t, tc.wantHasMore, res.HasMore)
			if tc.wantPartial {
				require.NotNil(t, res.PartialError)
			} else {
				require.Nil(t, res.PartialError)
			}
		})
	}
}

func TestPaginate_CancelledBeforeFirstPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var tokens []mo.Option[string]
	_, err := Paginate(ctx, mo.None[uint64](), "things", fetcher(nil, -1, &tokens), extract)
	require.Error(t, err)
	require.Empty(t, tokens)
}
