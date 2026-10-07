package command

import (
	"log/slog"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// ParsePaginationFlags reads the --page-size/--max-pages pair every list
// command shares. The max-pages flag is built without a lower bound so the -1
// "all pages" sentinel gets through, which leaves rejecting 0 and negatives to
// this switch.
func ParsePaginationFlags(
	pageSizeFlag, maxPagesFlag flags.IntegerFlag,
) (pageSize, maxPages mo.Option[uint64], err cenclierrors.CencliError) {
	rawPageSize, err := pageSizeFlag.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawPageSize.IsPresent() {
		pageSize = mo.Some(uint64(rawPageSize.MustGet()))
	}

	rawMaxPages, err := maxPagesFlag.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawMaxPages.IsPresent() {
		switch v := rawMaxPages.MustGet(); {
		case v == -1:
			maxPages = mo.None[uint64]()
		case v <= 0:
			return pageSize, maxPages, flags.NewIntegerFlagInvalidValueError("max-pages", v, "must be -1 or >= 1")
		default:
			maxPages = mo.Some(uint64(v))
		}
	}

	return pageSize, maxPages, nil
}

// WarnFetchingAllPages tells the user that --max-pages=-1 will keep requesting
// pages until the server runs out, since the call count is otherwise invisible
// until it is spent.
func WarnFetchingAllPages(quiet bool, logger *slog.Logger, maxPages mo.Option[uint64]) {
	if quiet || maxPages.IsPresent() {
		return
	}
	msg := styles.GlobalStyles.Warning.Render(
		"Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
	formatter.Println(formatter.Stderr, msg)
	logger.Debug("fetching all pages", "message", msg)
}
