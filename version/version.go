// Package version holds the build metadata that the release build (GoReleaser for
// the binaries, the Dockerfile for the image) stamps in through -ldflags -X.
// A plain go build keeps the defaults.
package version

// Build metadata, set at link time by the release build.
var (
	// Version is the release version without the leading "v", or "dev".
	Version = "dev"
	// Commit is the short git commit the binary was built from, or "none".
	Commit = "none"
	// Date is the build timestamp, or "unknown".
	Date = "unknown"
)

// String returns the build metadata as "Version (Commit) Date", the form the
// server logs at startup.
func String() string {
	return Version + " (" + Commit + ") " + Date
}
