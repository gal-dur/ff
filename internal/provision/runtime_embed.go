//go:build embedruntime

package provision

import _ "embed"

// The pinned llama.cpp archive, carried inside the binary and self-extracted into
// the cache on first run — so installing ff is one file, and the only download it
// ever makes is the model. The build fetches this file (gitignored) and verifies it
// against the pin before compiling; Runtime() verifies it again before extracting,
// so a build that bundled the wrong bytes refuses itself.
//
//go:embed runtime.tar.gz
var embeddedRuntime []byte
