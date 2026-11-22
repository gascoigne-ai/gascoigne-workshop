package request

import (
	"reflect"
	"testing"
)

func TestAnAccessSetRemembersAccessesByKind(t *testing.T) {
	run := Access{AccessCommand, "/bin/tool"}
	read := Access{AccessFile, "/bin/tool"}
	other := Access{AccessFile, "/etc/hosts"}
	cases := []struct {
		name     string
		build    func() AccessSet
		wantKeys []string
		has      []Access
		hasNot   []Access
	}{
		{"the zero value is empty", func() AccessSet { return AccessSet{} }, []string{}, nil, []Access{run}},
		{"an added access is there", func() AccessSet {
			var set AccessSet
			set.Add(run)
			return set
		}, []string{"command:/bin/tool"}, []Access{run}, []Access{other}},
		{"adding twice leaves one key", func() AccessSet {
			var set AccessSet
			set.Add(run)
			set.Add(run)
			return set
		}, []string{"command:/bin/tool"}, []Access{run}, nil},
		{"the same subject of another kind is another access", func() AccessSet {
			var set AccessSet
			set.Add(run)
			return set
		}, []string{"command:/bin/tool"}, []Access{run}, []Access{read}},
		{"keys come sorted", func() AccessSet {
			var set AccessSet
			set.Add(other)
			set.Add(run)
			set.Add(read)
			return set
		}, []string{"command:/bin/tool", "file:/bin/tool", "file:/etc/hosts"}, []Access{run, read, other}, nil},
		{"a set built from keys has them", func() AccessSet {
			return AccessSetOf([]string{"file:/etc/hosts", "command:/bin/tool", "file:/etc/hosts"})
		}, []string{"command:/bin/tool", "file:/etc/hosts"}, []Access{run, other}, []Access{read}},
		{"no keys is the empty set", func() AccessSet { return AccessSetOf(nil) }, []string{}, nil, []Access{run}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set := tc.build()
			if got := set.Keys(); !reflect.DeepEqual(got, tc.wantKeys) {
				t.Errorf("Keys = %v, want %v", got, tc.wantKeys)
			}
			for _, access := range tc.has {
				if !set.Has(access) {
					t.Errorf("Has(%v) = false", access)
				}
			}
			for _, access := range tc.hasNot {
				if set.Has(access) {
					t.Errorf("Has(%v) = true", access)
				}
			}
		})
	}
}

func TestAnAccessSetRoundTripsThroughItsKeys(t *testing.T) {
	var set AccessSet
	set.Add(Access{AccessFile, "/a"})
	set.Add(Access{AccessCommand, "b"})
	again := AccessSetOf(set.Keys())
	if !reflect.DeepEqual(again.Keys(), set.Keys()) || !again.Has(Access{AccessFile, "/a"}) {
		t.Fatalf("%v became %v", set.Keys(), again.Keys())
	}
}
