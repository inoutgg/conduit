package buildinfo

import "runtime/debug"

// BuildInfo exposes build-time metadata about the conduit binary.
type BuildInfo interface {
	Version() string
}

// Standard is the default BuildInfo implementation that reads version
// information from ldflags or the Go module build info.
type Standard struct{}

// buildVersion can be overridden at build time via:
//
//	go build -ldflags "-X go.inout.gg/conduit/pkg/buildinfo.buildVersion=v1.2.3"
//
//nolint:gochecknoglobals
var buildVersion string

// Version returns the conduit version. It prefers the value set via -ldflags,
// falls back to the Go module version from build info, and defaults to "devel".
func (Standard) Version() string {
	if buildVersion != "" {
		return buildVersion
	}

	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}

	return "devel"
}
