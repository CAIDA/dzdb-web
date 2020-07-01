// Package version contains build version information
package version

import "fmt"

// Git version variables
var (
	GitDate   = "?"
	GitHash   = "?"
	GitBranch = "?"
)

// String returns the version string
func String() string {
	return fmt.Sprintf("%s/%s (%s)", GitBranch, GitHash, GitDate)
}
