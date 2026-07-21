package tags

import (
	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

// requireTagID builds a TagID from a positional argument and rejects an empty
// (or whitespace-only) identifier, which no tag command can act on. Shared by
// get, update, and delete so the check is applied uniformly at the boundary.
func requireTagID(raw string) (identifiers.TagID, cenclierrors.CencliError) {
	id := identifiers.NewTagID(raw)
	if id.String() == "" {
		return id, tags.NewEmptyTagIDError()
	}
	return id, nil
}
