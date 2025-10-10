package assets

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// InvalidAssetIDError represents an error that occurs when an invalid asset ID is provided.
type InvalidAssetIDError interface {
	cenclierrors.CencliError
}

type invalidAssetIDError struct {
	assetID string
	reason  string
}

var _ InvalidAssetIDError = &invalidAssetIDError{}

func NewInvalidAssetIDError(assetID string, reason string) InvalidAssetIDError {
	return &invalidAssetIDError{assetID: assetID, reason: reason}
}

func (e *invalidAssetIDError) Error() string {
	return fmt.Sprintf("invalid asset ID: %s (%s)", e.assetID, e.reason)
}

func (e *invalidAssetIDError) Title() string {
	return "Invalid Asset ID"
}

func (e *invalidAssetIDError) ShouldPrintUsage() bool {
	return true
}

// NoAssetsError represents an error that occurs when no assets are provided.
type NoAssetsError interface {
	cenclierrors.CencliError
}

type noAssetsError struct{}

var _ NoAssetsError = &noAssetsError{}

func NewNoAssetsError() NoAssetsError {
	return &noAssetsError{}
}

func (e *noAssetsError) Title() string {
	return "No Assets Provided"
}

func (e *noAssetsError) Error() string { return "you must provide at least one asset" }

func (e *noAssetsError) ShouldPrintUsage() bool {
	return true
}

// MixedAssetTypesError represents an error that occurs when mixed asset types are provided.
type MixedAssetTypesError interface {
	cenclierrors.CencliError
}

type mixedAssetTypesError struct {
	typesFound []AssetType
}

func NewMixedAssetTypesError(typesFound ...AssetType) MixedAssetTypesError {
	return &mixedAssetTypesError{typesFound: typesFound}
}

func (e *mixedAssetTypesError) Title() string {
	return "Mixed Asset Types"
}

func (e *mixedAssetTypesError) Error() string {
	var output strings.Builder
	output.WriteString("mixed asset types: ")
	typesFound := make([]string, len(e.typesFound))
	for i, t := range e.typesFound {
		typesFound[i] = string(t)
	}
	output.WriteString(strings.Join(typesFound, ", "))
	return output.String()
}

func (e *mixedAssetTypesError) ShouldPrintUsage() bool {
	return true
}

type TooManyAssetsError interface {
	cenclierrors.CencliError
}

type tooManyAssetsError struct {
	provided  int
	supported int
}

func NewTooManyAssetsError(provided int, supported int) TooManyAssetsError {
	return &tooManyAssetsError{provided: provided, supported: supported}
}

func (e *tooManyAssetsError) Error() string {
	return fmt.Sprintf("%d assets provided, only %d are supported", e.provided, e.supported)
}

func (e *tooManyAssetsError) Title() string { return "Too Many Assets" }

func (e *tooManyAssetsError) ShouldPrintUsage() bool { return true }
