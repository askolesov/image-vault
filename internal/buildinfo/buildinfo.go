package buildinfo

import "fmt"

var (
	version    = "dev"
	commitHash = "unknown"
	buildDate  = "unknown"
	branch     = "unknown"
)

func Version() string    { return version }
func CommitHash() string { return commitHash }
func BuildDate() string  { return buildDate }
func Branch() string     { return branch }

func FullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, branch: %s)", version, commitHash, buildDate, branch)
}
