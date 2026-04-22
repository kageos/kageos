package service

import "testing"

func TestNormalizeRunTableBatchCreateBody(t *testing.T) {
	body, err := normalizeRunTableBatchCreateBody(`{"data":[{"title":"A"},{"title":"B"}]}`)
	if err != nil {
		t.Fatalf("normalize body returned error: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(body.Data))
	}
	if body.Data[0]["title"] != "A" {
		t.Fatalf("first title = %#v", body.Data[0]["title"])
	}
}

func TestNormalizeRunTableBatchCreateBodyRejectsInvalidBody(t *testing.T) {
	cases := []string{
		`[]`,
		`{"data":[]}`,
		`{"data":[null]}`,
		`{"data":["bad"]}`,
	}
	for _, tc := range cases {
		if _, err := normalizeRunTableBatchCreateBody(tc); err == nil {
			t.Fatalf("normalize body should reject %s", tc)
		}
	}
}
