package assets

import (
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// AssetType represents the inferred type for a given collection of user-supplied assets.
// Supported values are "host", "certificate", and "webproperty".
type AssetType string

const (
	AssetTypeUnknown     AssetType = "unknown"
	AssetTypeHost        AssetType = "host"
	AssetTypeCertificate AssetType = "certificate"
	AssetTypeWebProperty AssetType = "webproperty"
)

func (a AssetType) String() string { return string(a) }

// AssetClassifier classifies raw string inputs into typed asset identifiers and reports errors.
// It also deduplicates values within each asset category.
type AssetClassifier struct {
	hostIDs        map[HostID]struct{}
	certificateIDs map[CertificateID]struct{}
	webPropertyIDs map[WebPropertyID]struct{}
	unknownAssets  map[string]struct{}
	// preserve first-seen order for stable behavior and UX
	hostOrder        []HostID
	certificateOrder []CertificateID
	webPropertyOrder []WebPropertyID
	unknownOrder     []string
}

// NewAssetClassifier creates a classifier and immediately classifies the provided raw assets.
func NewAssetClassifier(rawAssets ...string) *AssetClassifier {
	a := &AssetClassifier{
		hostIDs:        make(map[HostID]struct{}),
		certificateIDs: make(map[CertificateID]struct{}),
		webPropertyIDs: make(map[WebPropertyID]struct{}),
		unknownAssets:  make(map[string]struct{}),
	}
	a.classify(rawAssets...)
	return a
}

// classify classifies the assets into their respective types.
func (a *AssetClassifier) classify(rawAssets ...string) {
	for _, arg := range rawAssets {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		if h, err := NewHostID(arg); err == nil {
			if _, exists := a.hostIDs[h]; !exists {
				a.hostIDs[h] = struct{}{}
				a.hostOrder = append(a.hostOrder, h)
			}
			continue
		}
		if c, err := NewCertificateFingerprint(arg); err == nil {
			if _, exists := a.certificateIDs[c]; !exists {
				a.certificateIDs[c] = struct{}{}
				a.certificateOrder = append(a.certificateOrder, c)
			}
			continue
		}
		if w, err := NewWebPropertyID(arg, DefaultWebPropertyPort); err == nil {
			if _, exists := a.webPropertyIDs[w]; !exists {
				a.webPropertyIDs[w] = struct{}{}
				a.webPropertyOrder = append(a.webPropertyOrder, w)
			}
			continue
		}
		if _, exists := a.unknownAssets[arg]; !exists {
			a.unknownAssets[arg] = struct{}{}
			a.unknownOrder = append(a.unknownOrder, arg)
		}
	}
}

// KnownAssetCount returns the number of known asset types that were passed to the classifier.
func (a *AssetClassifier) KnownAssetCount() int {
	return len(a.hostIDs) + len(a.certificateIDs) + len(a.webPropertyIDs)
}

// KnownAssetIDs returns the IDs of the known asset types that were passed to the classifier.
func (a *AssetClassifier) KnownAssetIDs() []string {
	res := make([]string, 0, a.KnownAssetCount())
	for _, h := range a.hostOrder {
		res = append(res, h.String())
	}
	for _, c := range a.certificateOrder {
		res = append(res, c.String())
	}
	for _, w := range a.webPropertyOrder {
		res = append(res, w.String())
	}
	return res
}

// AssetType returns the type of assets that were passed to the classifier.
// If no assets were passed, it returns AssetTypeUnknown.
// If multiple asset types were passed, it returns a MixedAssetTypesError.
func (a *AssetClassifier) AssetType() (AssetType, cenclierrors.CencliError) {
	unknowns := a.unknownOrder
	if len(unknowns) > 0 {
		return AssetTypeUnknown, NewInvalidAssetIDError(unknowns[0], "unable to infer asset type")
	}

	t := AssetTypeUnknown
	if len(a.hostIDs) > 0 {
		t = AssetTypeHost
	}
	if len(a.certificateIDs) > 0 {
		if t != AssetTypeUnknown {
			return AssetTypeUnknown, NewMixedAssetTypesError(t, AssetTypeCertificate)
		}
		t = AssetTypeCertificate
	}
	if len(a.webPropertyIDs) > 0 {
		if t != AssetTypeUnknown {
			return AssetTypeUnknown, NewMixedAssetTypesError(t, AssetTypeWebProperty)
		}
		t = AssetTypeWebProperty
	}
	if t == AssetTypeUnknown {
		return t, NewNoAssetsError()
	}
	return t, nil
}

func (a *AssetClassifier) HostIDs() []HostID {
	return append([]HostID(nil), a.hostOrder...)
}

func (a *AssetClassifier) CertificateIDs() []CertificateID {
	return append([]CertificateID(nil), a.certificateOrder...)
}

func (a *AssetClassifier) WebPropertyIDs() []WebPropertyID {
	return append([]WebPropertyID(nil), a.webPropertyOrder...)
}

func (a *AssetClassifier) UnknownAssets() []string {
	return append([]string(nil), a.unknownOrder...)
}
