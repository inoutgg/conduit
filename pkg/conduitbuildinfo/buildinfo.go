package conduitbuildinfo

import "runtime/debug"

// BuildInfo exposes build-time metadata about the conduit binary.
type BuildInfo interface {
	Version() string
}

// Standard reads version from ldflags or the Go module build info.
// Falls back to "devel" when neither is available.
type Standard struct{}

// buildVersion can be overridden at build time via:
//
//	go build -ldflags "-X go.inout.gg/conduit/pkg/conduitbuildinfo.buildVersion=v1.2.3"
//
//nolint:gochecknoglobals
var buildVersion string

// Version returns the conduit version.
func (Standard) Version() string {
	if buildVersion != "" {
		return buildVersion
	}

	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}

	return "devel"
}

// Stub is a BuildInfo implementation that returns a fixed version string.
// This is intended for use in tests.
type Stub struct{ V string }

func (s Stub) Version() string { return s.V }
