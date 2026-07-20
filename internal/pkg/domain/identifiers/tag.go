package identifiers

import (
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"
)

// TagID identifies a tag by either its name or its UUID. The raw value is
// preserved verbatim and passed straight through to the tag_id path parameter.
// Only the GetTag endpoint accepts a name there; the mutate endpoints
// (update, delete) require a UUID and reject a name with a 422. UID reports the
// UUID when the raw value parses as one.
type TagID struct{ raw string }

// NewTagID builds a TagID from a raw name-or-UUID string, trimming surrounding
// whitespace.
func NewTagID(raw string) TagID {
	return TagID{raw: strings.TrimSpace(raw)}
}

// String returns the raw name-or-UUID value, suitable for the API path parameter.
func (t TagID) String() string { return t.raw }

// UID returns the parsed UUID when the raw value is a valid UUID, otherwise None
// (i.e. the value is a tag name).
func (t TagID) UID() mo.Option[uuid.UUID] {
	if u, err := uuid.Parse(t.raw); err == nil {
		return mo.Some(u)
	}
	return mo.None[uuid.UUID]()
}
