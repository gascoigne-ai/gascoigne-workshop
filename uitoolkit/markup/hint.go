package markup

import (
	"fmt"
	"strings"
)

// hint is the end of an error message about an unknown name: the
// closest known name, when one is near, and the list of all of them.
func hint(got string, known []string, what string) string {
	best, bestD := "", 3
	for _, k := range known {
		if d := distance(got, k); d < bestD {
			best, bestD = k, d
		}
	}
	out := ""
	if best != "" {
		out = fmt.Sprintf(" (did you mean %q?)", best)
	}
	return out + "; " + what + ": " + strings.Join(known, " ")
}

// distance is the edit distance between a and b.
func distance(first, second string) int {
	prev := make([]int, len(second)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(first); i++ {
		cur := make([]int, len(second)+1)
		cur[0] = i
		for j := 1; j <= len(second); j++ {
			cost := 1
			if first[i-1] == second[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(second)]
}
