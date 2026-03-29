package plugin

import (
	"strconv"
	"strings"
)

// compareSemver compares two dotted-numeric version strings (e.g. "1.2.3").
// It returns -1 if a < b, 0 if a == b, and 1 if a > b. Missing segments are
// treated as zero, so "1.2" equals "1.2.0".
func compareSemver(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}

	for i := 0; i < max; i++ {
		av := 0
		bv := 0
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

