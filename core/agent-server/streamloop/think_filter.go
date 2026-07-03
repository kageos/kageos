package streamloop

import "strings"

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

type thinkTagFilter struct {
	inThink bool
	pending string
	saw     bool
}

type thinkFilterChunk struct {
	Content  string
	Thinking string
}

func newThinkTagFilter() *thinkTagFilter {
	return &thinkTagFilter{}
}

func (f *thinkTagFilter) Append(chunk string) thinkFilterChunk {
	if chunk == "" {
		return thinkFilterChunk{}
	}
	input := f.pending + chunk
	f.pending = ""

	var out strings.Builder
	var thinking strings.Builder
	for input != "" {
		if f.inThink {
			idx := indexFold(input, thinkCloseTag)
			if idx < 0 {
				f.pending = suffixThatPrefixes(input, thinkCloseTag)
				thinking.WriteString(input[:len(input)-len(f.pending)])
				return thinkFilterChunk{Content: out.String(), Thinking: thinking.String()}
			}
			f.saw = true
			thinking.WriteString(input[:idx])
			input = input[idx+len(thinkCloseTag):]
			f.inThink = false
			continue
		}

		openIdx := indexFold(input, thinkOpenTag)
		closeIdx := indexFold(input, thinkCloseTag)
		if closeIdx >= 0 && (openIdx < 0 || closeIdx < openIdx) {
			out.WriteString(input[:closeIdx])
			f.saw = true
			input = input[closeIdx+len(thinkCloseTag):]
			continue
		}
		if openIdx < 0 {
			keep := suffixThatPrefixes(input, thinkOpenTag)
			out.WriteString(input[:len(input)-len(keep)])
			f.pending = keep
			return thinkFilterChunk{Content: out.String(), Thinking: thinking.String()}
		}
		out.WriteString(input[:openIdx])
		f.saw = true
		input = input[openIdx+len(thinkOpenTag):]
		f.inThink = true
	}
	return thinkFilterChunk{Content: out.String(), Thinking: thinking.String()}
}

func (f *thinkTagFilter) Finish() string {
	if f.inThink {
		f.pending = ""
		return ""
	}
	tail := f.pending
	f.pending = ""
	return tail
}

func (f *thinkTagFilter) SawThink() bool {
	return f.saw
}

func indexFold(s, substr string) int {
	if substr == "" {
		return 0
	}
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}

func suffixThatPrefixes(s, prefix string) string {
	lowerS := strings.ToLower(s)
	lowerPrefix := strings.ToLower(prefix)
	max := len(lowerPrefix) - 1
	if max > len(lowerS) {
		max = len(lowerS)
	}
	for n := max; n > 0; n-- {
		if strings.HasSuffix(lowerS, lowerPrefix[:n]) {
			return s[len(s)-n:]
		}
	}
	return ""
}
