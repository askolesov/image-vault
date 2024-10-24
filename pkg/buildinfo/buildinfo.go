package buildinfo

import (
	"runtime"

	"gopkg.in/yaml.v3"
)

// Provisioned by ldflags
var (
	version    string
	branch     string
	commitHash string
	buildDate  string
)

type BuildInfo struct {
	Version    string `json:"version"`
	Branch     string `json:"branch"`
	CommitHash string `json:"commit_hash"`
	BuildDate  string `json:"build_date"`
	GoVersion  string `json:"go_version"`
	GoOS       string `json:"go_os"`
	GoArch     string `json:"go_arch"`
	Compiler   string `json:"compiler"`
}

func Get() *BuildInfo {
	return &BuildInfo{
		Version:    version,
		Branch:     branch,
		CommitHash: commitHash,
		BuildDate:  buildDate,
		GoVersion:  runtime.Version(),
		GoOS:       runtime.GOOS,
		GoArch:     runtime.GOARCH,
		Compiler:   runtime.Compiler,
	}
}

// YAML returns build info in compressed yaml format
func (b *BuildInfo) YAML() []byte {
	res, err := yaml.Marshal(b) //nolint: musttag
	if err != nil {
		panic(err) // should never happen
	}

	return res
}
