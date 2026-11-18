package model

import (
	"maps"
	"slices"
)

// AccessSet is a set of Accesses, kept by Key. The zero value is an empty
// set that is ready to use.
type AccessSet struct {
	keys map[string]struct{}
}

// AccessSetOf is the set of the remembered keys, as a state file lists them.
func AccessSetOf(keys []string) AccessSet {
	var set AccessSet
	for _, key := range keys {
		set.addKey(key)
	}
	return set
}

// Has reports whether access is in the set.
func (set AccessSet) Has(access Access) bool {
	_, ok := set.keys[access.Key()]
	return ok
}

// Add puts access into the set.
func (set *AccessSet) Add(access Access) { set.addKey(access.Key()) }

func (set *AccessSet) addKey(key string) {
	if set.keys == nil {
		set.keys = map[string]struct{}{}
	}
	set.keys[key] = struct{}{}
}

// Keys are the keys of the set in sorted order, never nil.
func (set AccessSet) Keys() []string {
	keys := slices.AppendSeq(make([]string, 0, len(set.keys)), maps.Keys(set.keys))
	slices.Sort(keys)
	return keys
}
