package v2

import (
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/samber/lo"
)

func FilterIgnore(
	paths []string,
	ignorePatterns []string,
) []string {
	obj := ignore.CompileIgnoreLines(ignorePatterns...)

	return lo.Filter(paths, func(path string, _ int) bool {
		return !obj.MatchesPath(path)
	})
}
