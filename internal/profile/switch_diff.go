package profile

import "github.com/larah/nd/internal/nd"

// SwitchDiff represents the computed difference between two profiles
// for the profile switch algorithm (spec: profile scope semantics).
type SwitchDiff struct {
	// Keep contains assets in both profiles with matching four-tuple.
	Keep []ProfileAsset

	// Remove contains assets only in the current profile (to be removed).
	Remove []ProfileAsset

	// Deploy contains assets only in the target profile (to be deployed).
	Deploy []ProfileAsset
}

// assetKey is the four-tuple equality key for profile switch diff.
type assetKey struct {
	SourceID  string
	AssetType nd.AssetType
	AssetName string
	Scope     nd.Scope
}

func keyOf(pa ProfileAsset) assetKey {
	return assetKey{
		SourceID:  pa.SourceID,
		AssetType: pa.AssetType,
		AssetName: pa.AssetName,
		Scope:     pa.Scope,
	}
}

// ComputeSwitchDiff computes the diff between current and target profiles.
// Equality is determined by (source_id, asset_type, asset_name, scope).
func ComputeSwitchDiff(current, target *Profile) SwitchDiff {
	targetSet := make(map[assetKey]ProfileAsset, len(target.Assets))
	for _, a := range target.Assets {
		targetSet[keyOf(a)] = a
	}

	matched := make(map[assetKey]bool)
	var diff SwitchDiff

	for _, a := range current.Assets {
		k := keyOf(a)
		if _, ok := targetSet[k]; ok {
			diff.Keep = append(diff.Keep, a)
			matched[k] = true
		} else {
			diff.Remove = append(diff.Remove, a)
		}
	}

	for _, a := range target.Assets {
		if !matched[keyOf(a)] {
			diff.Deploy = append(diff.Deploy, a)
		}
	}

	return diff
}
