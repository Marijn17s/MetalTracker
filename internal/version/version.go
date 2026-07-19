package version

// Version is the app semver (with optional leading "v"), set at link time:
//
//	go build -ldflags "-X MetalTracker/internal/version.Version=v1.2.0"
//
// Local/dev builds keep the default "dev".
var Version = "dev"

// Display returns a UI-friendly version string.
func Display() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
