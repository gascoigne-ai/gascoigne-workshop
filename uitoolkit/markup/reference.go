package markup

import (
	"maps"
	"slices"
	"sort"
)

func elementNames() []string {
	names := slices.Sorted(maps.Keys(elements))
	return names
}

func attributeNames() []string {
	names := make([]string, 0, len(attributes)+len(directives)+len(handlers))
	for n := range attributes {
		names = append(names, n)
	}
	for n := range directives {
		names = append(names, n)
	}
	for n := range handlers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
