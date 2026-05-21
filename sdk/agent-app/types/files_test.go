package types

import "testing"

func TestFilesStringProtocol(t *testing.T) {
	refs := NormalizeFileRefs([]string{
		"/workspace/a.png",
		"workspace/a.png",
		"workspace/b.pdf",
		"",
	})

	if got, want := JoinFileRefs(refs), "workspace/a.png,workspace/b.pdf"; got != want {
		t.Fatalf("JoinFileRefs() = %q, want %q", got, want)
	}

	parsed := ParseFileRefs("workspace/a.png,workspace/b.pdf")
	if got, want := parsed, []string{"workspace/a.png", "workspace/b.pdf"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ParseFileRefs() = %#v, want %#v", got, want)
	}
}

func TestSplitAndJoinFileRef(t *testing.T) {
	bucket, key := SplitFileRef("/kageos/workspace/chat/a.png")
	if bucket != "kageos" || key != "workspace/chat/a.png" {
		t.Fatalf("SplitFileRef() = %q, %q", bucket, key)
	}
	if got, want := JoinFileRef(bucket, key), "kageos/workspace/chat/a.png"; got != want {
		t.Fatalf("JoinFileRef() = %q, want %q", got, want)
	}
}
