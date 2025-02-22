package vault

import (
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/samber/lo"
)

func FilterIgnore(
	paths []string,
	ignorePatterns []string,
	progressCb func(int64),
) []string {
	obj := ignore.CompileIgnoreLines(ignorePatterns...)

	return lo.Filter(paths, func(path string, _ int) bool {
		if progressCb != nil {
			progressCb(1)
		}
		return !obj.MatchesPath(path)
	})
}
