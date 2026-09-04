// Command pin prints the artifact pins in shell-eval form, for the Makefile and the
// release workflow. The pins live in internal/pin and nowhere else; this is how a
// shell reads them without a second copy to drift.
package main

import (
	"fmt"

	"github.com/gal-dur/ff/internal/pin"
)

func main() {
	fmt.Printf("RUNTIME_URL=%q\n", pin.RuntimeURL)
	fmt.Printf("RUNTIME_SHA256=%q\n", pin.RuntimeSHA256)
}
