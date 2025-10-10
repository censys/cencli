package view

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

type AtTimeNotSupportedError interface {
	cenclierrors.CencliError
}

type atTimeNotSupportedError struct {
	assetType assets.AssetType
}

func NewAtTimeNotSupportedError(assetType assets.AssetType) AtTimeNotSupportedError {
	return &atTimeNotSupportedError{assetType: assetType}
}

func (e *atTimeNotSupportedError) Error() string {
	return fmt.Sprintf("at-time is not supported for %s assets", e.assetType)
}

func (e *atTimeNotSupportedError) Title() string {
	return "At-Time Not Supported"
}

func (e *atTimeNotSupportedError) ShouldPrintUsage() bool {
	return true
}

// UnsupportedAssetTypeError indicates that a provided asset type is not supported by the view command.
type UnsupportedAssetTypeError interface {
	cenclierrors.CencliError
}

type unsupportedAssetTypeError struct {
	assetType assets.AssetType
	reason    string
}

func NewUnsupportedAssetTypeError(assetType assets.AssetType, reason string) UnsupportedAssetTypeError {
	return &unsupportedAssetTypeError{assetType: assetType, reason: reason}
}

func (e *unsupportedAssetTypeError) Error() string {
	return fmt.Sprintf("unsupported asset type: %s (%s)", e.assetType, e.reason)
}

func (e *unsupportedAssetTypeError) Title() string {
	return "Unsupported Asset Type"
}

func (e *unsupportedAssetTypeError) ShouldPrintUsage() bool {
	return true
}
