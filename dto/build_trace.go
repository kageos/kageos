package dto

// BuildTrace records the observable timing profile for one workspace build/update.
type BuildTrace struct {
	TraceID     string           `json:"trace_id,omitempty"`
	Operation   string           `json:"operation,omitempty"`
	User        string           `json:"user,omitempty"`
	App         string           `json:"app,omitempty"`
	StartedAt   string           `json:"started_at,omitempty"`
	EndedAt     string           `json:"ended_at,omitempty"`
	DurationMS  int64            `json:"duration_ms,omitempty"`
	Status      string           `json:"status,omitempty"`
	Error       string           `json:"error,omitempty"`
	StoragePath string           `json:"storage_path,omitempty"`
	Spans       []BuildTraceSpan `json:"spans,omitempty"`
}

// BuildTraceSpan records one timed phase in the build/update flow.
type BuildTraceSpan struct {
	Seq        int               `json:"seq,omitempty"`
	Name       string            `json:"name,omitempty"`
	StartedAt  string            `json:"started_at,omitempty"`
	EndedAt    string            `json:"ended_at,omitempty"`
	DurationMS int64             `json:"duration_ms,omitempty"`
	Status     string            `json:"status,omitempty"`
	Error      string            `json:"error,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}
