// Command pin is the build scripts' window into internal/pin, so the Makefile and
// the release workflow can never disagree with the code.
//
//	pin platforms            print the pinned GOOS/GOARCH pairs, one per line
//	pin fetch [goos goarch]  fetch and verify that platform's llama.cpp archive
//	                         into internal/provision/runtime.tar.gz (default: this
//	                         machine's platform), reusing provision's resumable,
//	                         checksum-gated download — the shell holds no copy of
//	                         the fetch-then-verify dance.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/gal-dur/ff/internal/pin"
	"github.com/gal-dur/ff/internal/provision"
)

const blob = "internal/provision/runtime.tar.gz"

func fail(err error) {
	fmt.Fprintln(os.Stderr, "pin:", err)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]
	switch {
	case len(args) == 1 && args[0] == "platforms":
		for _, platform := range pin.Platforms() {
			fmt.Println(platform)
		}
	case (len(args) == 1 || len(args) == 3) && args[0] == "fetch":
		goos, goarch := runtime.GOOS, runtime.GOARCH
		if len(args) == 3 {
			goos, goarch = args[1], args[2]
		}
		artifact, err := pin.RuntimeFor(goos, goarch)
		if err != nil {
			fail(err)
		}
		if err := provision.Fetch(artifact, blob); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("usage: pin platforms | pin fetch [goos goarch]"))
	}
}
