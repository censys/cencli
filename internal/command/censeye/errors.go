package censeye

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

type (
	ErrorAssetTypeNotSupportedError interface{ cenclierrors.CencliError }
	errorAssetTypeNotSupportedError struct {
		assetType assets.AssetType
	}
)

func newErrorAssetTypeNotSupportedError(assetType assets.AssetType) ErrorAssetTypeNotSupportedError {
	return &errorAssetTypeNotSupportedError{assetType: assetType}
}

func (e *errorAssetTypeNotSupportedError) Error() string {
	return fmt.Sprintf("asset type %s is not supported for this command", e.assetType)
}

func (e *errorAssetTypeNotSupportedError) Title() string {
	return "Asset Type Not Supported"
}
func (e *errorAssetTypeNotSupportedError) ShouldPrintUsage() bool { return true }

type (
	InvalidRarityFlagError interface{ cenclierrors.CencliError }
	invalidRarityFlagError struct {
		flagName string
		reason   string
	}
)

func newInvalidRarityFlagError(flagName, reason string) InvalidRarityFlagError {
	return &invalidRarityFlagError{flagName: flagName, reason: reason}
}

func (e *invalidRarityFlagError) Error() string {
	return fmt.Sprintf("invalid value for --%s: %s", e.flagName, e.reason)
}

func (e *invalidRarityFlagError) Title() string {
	return "Invalid Rarity Flag"
}
func (e *invalidRarityFlagError) ShouldPrintUsage() bool { return true }

// HostNotFoundError is returned when a host lookup returns no results.
type (
	HostNotFoundError interface{ cenclierrors.CencliError }
	hostNotFoundError struct {
		hostID string
	}
)

func newHostNotFoundError(hostID string) HostNotFoundError { return &hostNotFoundError{hostID: hostID} }

func (e *hostNotFoundError) Error() string {
	return fmt.Sprintf("host %s not found", e.hostID)
}

func (e *hostNotFoundError) Title() string { return "Host Not Found" }

func (e *hostNotFoundError) ShouldPrintUsage() bool { return false }
