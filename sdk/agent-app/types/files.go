package types

import "strings"

// ParseFileRefs parses the files widget string protocol.
//
// A files field is persisted and transported as "bucket/object_key"; multiple
// files use comma separation.
func ParseFileRefs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return NormalizeFileRefs(strings.Split(value, ","))
}

func ParseRefs(value string) []string {
	return ParseFileRefs(value)
}

func NormalizeFileRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		ref = NormalizeFileRef(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func NormalizeRefs(refs []string) []string {
	return NormalizeFileRefs(refs)
}

func NormalizeFileRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "/")
	for strings.Contains(ref, "//") {
		ref = strings.ReplaceAll(ref, "//", "/")
	}
	return ref
}

func NormalizeRef(ref string) string {
	return NormalizeFileRef(ref)
}

func JoinFileRef(bucket, key string) string {
	bucket = NormalizeFileRef(bucket)
	key = NormalizeFileRef(key)
	if bucket == "" {
		return key
	}
	if key == "" {
		return bucket
	}
	return bucket + "/" + key
}

func JoinRef(bucket, key string) string {
	return JoinFileRef(bucket, key)
}

func SplitFileRef(ref string) (bucket string, key string) {
	ref = NormalizeFileRef(ref)
	if ref == "" {
		return "", ""
	}
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

func SplitRef(ref string) (bucket string, key string) {
	return SplitFileRef(ref)
}

func JoinFileRefs(refs []string) string {
	return strings.Join(NormalizeFileRefs(refs), ",")
}
