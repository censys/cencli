package identifiers

import (
	"github.com/google/uuid"
)

// CollectionID represents a validated collection UUID.
type CollectionID struct{ value uuid.UUID }

func (i CollectionID) String() string { return i.value.String() }

func NewCollectionID(uuid uuid.UUID) CollectionID {
	return CollectionID{value: uuid}
}
