package service

import "testing"

func TestNormalizeRunTableDeleteIDs(t *testing.T) {
	ids, err := normalizeRunTableDeleteIDs([]interface{}{float64(1), "2", int64(3), int(4)})
	if err != nil {
		t.Fatalf("normalize ids returned error: %v", err)
	}
	want := []int64{1, 2, 3, 4}
	if len(ids) != len(want) {
		t.Fatalf("ids len = %d, want %d", len(ids), len(want))
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids[%d] = %d, want %d", i, ids[i], want[i])
		}
	}
}

func TestNormalizeRunTableDeleteIDsRejectsInvalidValues(t *testing.T) {
	cases := [][]interface{}{
		{},
		{float64(1.2)},
		{float64(0)},
		{"abc"},
		{map[string]interface{}{"id": 1}},
	}
	for _, tc := range cases {
		if _, err := normalizeRunTableDeleteIDs(tc); err == nil {
			t.Fatalf("normalize ids should reject %#v", tc)
		}
	}
}
