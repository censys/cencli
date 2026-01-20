package identifiers

import (
	"github.com/google/uuid"
)

// OrganizationID represents a validated organization UUID.
type OrganizationID struct{ value uuid.UUID }

func (i OrganizationID) String() string { return i.value.String() }

func (i OrganizationID) IsZero() bool { return i.value == uuid.Nil }

func NewOrganizationID(uuid uuid.UUID) OrganizationID {
	return OrganizationID{value: uuid}
}
