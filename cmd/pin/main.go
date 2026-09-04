// Command pin prints the artifact pins in shell-eval form, for the Makefile and the
// release workflow. The pins live in internal/pin and nowhere else; this is how a
// shell reads them without a second copy to drift.
//
// The target platform comes as arguments — `pin <goos> <goarch>` — so a build host
// can ask about any platform it cross-compiles for; with no arguments it answers
// for itself.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/gal-dur/ff/internal/pin"
)

func main() {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if len(os.Args) == 3 {
		goos, goarch = os.Args[1], os.Args[2]
	} else if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: pin [goos goarch]")
		os.Exit(2)
	}
	r, err := pin.RuntimeFor(goos, goarch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pin:", err)
		os.Exit(1)
	}
	fmt.Printf("RUNTIME_URL=%q\n", r.URL)
	fmt.Printf("RUNTIME_SHA256=%q\n", r.SHA256)
}
