package version

import "fmt"

var (
	GitDate   = "?"
	GitHash   = "?"
	GitBranch = "?"
)

// String returns the version string
func String() string {
	return fmt.Sprintf("%s/%s (%s)", GitBranch, GitHash, GitDate)
}
