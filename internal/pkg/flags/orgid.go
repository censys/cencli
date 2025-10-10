package flags

import (
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

const (
	orgIDFlagName  = "org-id"
	orgIDFlagShort = "o"
	orgIDFlagDesc  = "override the configured organization ID"
)

// OrgIDFlag is a domain-specific flag that represents an optional Organization ID.
type OrgIDFlag interface {
	// Value returns an optional value indicating the current value of the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	// If the flag has an invalid UUID, it returns an error of type InvalidUUIDFlagError.
	// An optional value is returned to keep callers from having to compare to uuid.Nil.
	Value() (mo.Option[identifiers.OrganizationID], cenclierrors.CencliError)
}

// NewOrgIDFlag instantiates a new OrgIDFlag on a given flag set.
// Essentially the same as a UUIDFlag, but has a defined flag name and description.
func NewOrgIDFlag(flags *pflag.FlagSet, shortOverride string) OrgIDFlag {
	short := orgIDFlagShort
	if shortOverride != "" {
		short = shortOverride
	}
	uuidF := NewUUIDFlag(flags, false, orgIDFlagName, short, mo.None[uuid.UUID](), orgIDFlagDesc)
	return &orgIDFlag{uuidFlag: uuidF}
}

type orgIDFlag struct {
	*uuidFlag
}

var _ OrgIDFlag = (*orgIDFlag)(nil)

func (f *orgIDFlag) Value() (mo.Option[identifiers.OrganizationID], cenclierrors.CencliError) {
	uid, err := f.uuidFlag.Value()
	if err != nil {
		return mo.None[identifiers.OrganizationID](), err
	}
	if uid.IsPresent() {
		return mo.Some(identifiers.NewOrganizationID(uid.MustGet())), nil
	}
	return mo.None[identifiers.OrganizationID](), nil
}
