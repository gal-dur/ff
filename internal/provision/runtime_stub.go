//go:build !embedruntime

package provision

// No runtime is bundled in this build — tests and ad-hoc `go build` fall back to
// downloading the pinned archive. Release builds carry it: see runtime_embed.go and
// the `embedruntime` tag the Makefile and workflow set.
var embeddedRuntime []byte
