// Command p2p sends files directly between two computers, end-to-end encrypted.
//
// See SPEC.md for what it does and docs/adr/ for why it is built this way.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is replaced at release time: go build -ldflags "-X main.version=v0.1.0".
var version = "dev"

const usageText = `p2p — encrypted peer-to-peer file sending

Usage:
  p2p send <path>     send a file or folder; prints a code to give the receiver
  p2p receive         receive files (asks for the code)
  p2p share <path>    print a link anyone can download from (swarm)
  p2p get <link>      download from a share link
  p2p relay           run a public relay / bootstrap node
  p2p version         print the version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run holds all of main's logic and returns the process exit code.
// Taking args and writers as parameters (instead of using os.Args and
// os.Stdout directly) is what makes it testable without starting a process.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "p2p", version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	case "send", "receive", "share", "get", "relay":
		fmt.Fprintf(stderr, "p2p %s: not implemented yet (see SPEC.md §5 for the roadmap)\n", args[0])
		return 1
	default:
		fmt.Fprintf(stderr, "p2p: unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}
