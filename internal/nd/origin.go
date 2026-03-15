package nd

// DeployOrigin tracks how an asset was deployed.
type DeployOrigin string

const (
	OriginManual DeployOrigin = "manual"
	OriginPinned DeployOrigin = "pinned"
)

// OriginProfile returns a profile-scoped deploy origin.
func OriginProfile(name string) DeployOrigin {
	return DeployOrigin("profile:" + name)
}

// IsProfile returns true if this origin is a profile deployment.
func (o DeployOrigin) IsProfile() bool {
	return len(o) > 8 && o[:8] == "profile:"
}

// ProfileName extracts the profile name, or "" if not a profile origin.
func (o DeployOrigin) ProfileName() string {
	if o.IsProfile() {
		return string(o[8:])
	}
	return ""
}
