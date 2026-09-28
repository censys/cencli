package history

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

type invalidModeError struct {
	mode string
}

func newInvalidModeError(mode string) cenclierrors.CencliError {
	return &invalidModeError{mode: mode}
}

func (e *invalidModeError) Error() string {
	return fmt.Sprintf("invalid --mode %q: must be snapshots or events", e.mode)
}

func (e *invalidModeError) Title() string { return "Invalid Mode" }

func (e *invalidModeError) ShouldPrintUsage() bool { return true }

type modeNotApplicableError struct {
	assetType assets.AssetType
}

func newModeNotApplicableError(assetType assets.AssetType) cenclierrors.CencliError {
	return &modeNotApplicableError{assetType: assetType}
}

func (e *modeNotApplicableError) Error() string {
	return fmt.Sprintf("--mode only applies to web properties, not to a %s", e.assetType)
}

func (e *modeNotApplicableError) Title() string { return "Conflicting Flags" }

func (e *modeNotApplicableError) ShouldPrintUsage() bool { return true }
