// Package buildinfo carries the build version, injected at link time.
package buildinfo

// Version is set via -ldflags "-X turbokey/internal/buildinfo.Version=vX.Y.Z".
// It stays "dev" for plain local builds.
var Version = "dev"
