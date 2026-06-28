package buildtrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

const (
	statusOK      = "ok"
	statusError   = "error"
	statusRunning = "running"

	maxErrorLen = 4000
)

type contextKey struct{}

// Attr is a string attribute attached to a timing span.
type Attr struct {
	Key   string
	Value string
}

// String records a string attribute.
func String(key, value string) Attr {
	return Attr{Key: strings.TrimSpace(key), Value: value}
}

// Int records an integer attribute.
func Int(key string, value int) Attr {
	return Attr{Key: strings.TrimSpace(key), Value: strconv.Itoa(value)}
}

// Int64 records an int64 attribute.
func Int64(key string, value int64) Attr {
	return Attr{Key: strings.TrimSpace(key), Value: strconv.FormatInt(value, 10)}
}

// Bool records a boolean attribute.
func Bool(key string, value bool) Attr {
	return Attr{Key: strings.TrimSpace(key), Value: strconv.FormatBool(value)}
}

// Trace collects build/update spans. It is safe for concurrent span completion.
type Trace struct {
	mu       sync.Mutex
	summary  dto.BuildTrace
	start    time.Time
	nextSeq  int
	finished bool
}

// New creates a trace for one build/update operation.
func New(operation, user, app string) *Trace {
	now := time.Now().UTC()
	return &Trace{
		start: now,
		summary: dto.BuildTrace{
			TraceID:   newTraceID(now),
			Operation: strings.TrimSpace(operation),
			User:      strings.TrimSpace(user),
			App:       strings.TrimSpace(app),
			StartedAt: formatTime(now),
			Status:    statusRunning,
		},
	}
}

// WithTrace stores a trace in context so deeper helpers can record spans.
func WithTrace(ctx context.Context, trace *Trace) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if trace == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, trace)
}

// FromContext returns the trace attached to ctx.
func FromContext(ctx context.Context) (*Trace, bool) {
	if ctx == nil {
		return nil, false
	}
	trace, ok := ctx.Value(contextKey{}).(*Trace)
	return trace, ok && trace != nil
}

// Ensure returns the existing trace on ctx or creates and attaches a new one.
func Ensure(ctx context.Context, operation, user, app string) (context.Context, *Trace) {
	if trace, ok := FromContext(ctx); ok {
		return ctx, trace
	}
	trace := New(operation, user, app)
	return WithTrace(ctx, trace), trace
}

// Span is an in-flight timed phase.
type Span struct {
	ctx     context.Context
	trace   *Trace
	record  dto.BuildTraceSpan
	start   time.Time
	once    sync.Once
	enabled bool
}

// Start begins a timed span. If ctx has no trace, it returns a no-op span.
func Start(ctx context.Context, name string, attrs ...Attr) *Span {
	trace, ok := FromContext(ctx)
	if !ok {
		return &Span{}
	}

	now := time.Now().UTC()
	trace.mu.Lock()
	trace.nextSeq++
	seq := trace.nextSeq
	trace.mu.Unlock()

	return &Span{
		ctx:   ctx,
		trace: trace,
		start: now,
		record: dto.BuildTraceSpan{
			Seq:        seq,
			Name:       strings.TrimSpace(name),
			StartedAt:  formatTime(now),
			Status:     statusRunning,
			Attributes: attrsToMap(attrs),
		},
		enabled: true,
	}
}

// Finish completes the span and appends it to the trace.
func (s *Span) Finish(err error) {
	if s == nil || !s.enabled || s.trace == nil {
		return
	}
	s.once.Do(func() {
		now := time.Now().UTC()
		s.record.EndedAt = formatTime(now)
		s.record.DurationMS = durationMillis(now.Sub(s.start))
		if err != nil {
			s.record.Status = statusError
			s.record.Error = compactError(err)
		} else {
			s.record.Status = statusOK
		}

		s.trace.mu.Lock()
		s.trace.summary.Spans = append(s.trace.summary.Spans, cloneSpan(s.record))
		traceID := s.trace.summary.TraceID
		s.trace.mu.Unlock()

		if err != nil {
			logger.Errorf(s.ctx, "[BuildTrace] trace_id=%s span=%s status=%s duration_ms=%d error=%s attrs=%v",
				traceID, s.record.Name, s.record.Status, s.record.DurationMS, s.record.Error, s.record.Attributes)
			return
		}
		logger.Infof(s.ctx, "[BuildTrace] trace_id=%s span=%s status=%s duration_ms=%d attrs=%v",
			traceID, s.record.Name, s.record.Status, s.record.DurationMS, s.record.Attributes)
	})
}

// Finalize marks the trace done and returns a snapshot.
func (t *Trace) Finalize(err error) *dto.BuildTrace {
	if t == nil {
		return nil
	}
	now := time.Now().UTC()

	t.mu.Lock()
	if !t.finished {
		t.finished = true
		t.summary.EndedAt = formatTime(now)
		t.summary.DurationMS = durationMillis(now.Sub(t.start))
		if err != nil {
			t.summary.Status = statusError
			t.summary.Error = compactError(err)
		} else {
			t.summary.Status = statusOK
		}
	}
	snapshot := cloneTraceLocked(&t.summary)
	t.mu.Unlock()
	return snapshot
}

// Snapshot returns a copy of the current trace.
func (t *Trace) Snapshot() *dto.BuildTrace {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return cloneTraceLocked(&t.summary)
}

// Persist writes the trace as <trace_id>.json and latest.json in dir.
func Persist(trace *Trace, dir string) (string, error) {
	if trace == nil {
		return "", nil
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("build trace directory is empty")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	snapshot := trace.Snapshot()
	if snapshot == nil || snapshot.TraceID == "" {
		return "", fmt.Errorf("build trace is empty")
	}
	path := filepath.Join(dir, snapshot.TraceID+".json")
	snapshot.StoragePath = path

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), data, 0644); err != nil {
		return "", err
	}

	trace.setStoragePath(path)
	return path, nil
}

// Slowest returns the slowest spans from a trace snapshot.
func Slowest(trace *dto.BuildTrace, limit int) []dto.BuildTraceSpan {
	if trace == nil || limit <= 0 || len(trace.Spans) == 0 {
		return nil
	}
	spans := make([]dto.BuildTraceSpan, 0, len(trace.Spans))
	for _, span := range trace.Spans {
		spans = append(spans, cloneSpan(span))
	}
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].DurationMS == spans[j].DurationMS {
			return spans[i].Seq < spans[j].Seq
		}
		return spans[i].DurationMS > spans[j].DurationMS
	})
	if len(spans) > limit {
		spans = spans[:limit]
	}
	return spans
}

// Summary renders a compact timing summary for logs or user-facing tool output.
func Summary(trace *dto.BuildTrace, limit int) string {
	if trace == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("total=%dms", trace.DurationMS)}
	if trace.StoragePath != "" {
		parts = append(parts, "stored="+trace.StoragePath)
	}
	for _, span := range Slowest(trace, limit) {
		if span.Name == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%dms", span.Name, span.DurationMS))
	}
	return strings.Join(parts, ", ")
}

func (t *Trace) setStoragePath(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.summary.StoragePath = path
}

func cloneTraceLocked(in *dto.BuildTrace) *dto.BuildTrace {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Spans) > 0 {
		out.Spans = make([]dto.BuildTraceSpan, 0, len(in.Spans))
		for _, span := range in.Spans {
			out.Spans = append(out.Spans, cloneSpan(span))
		}
		sort.SliceStable(out.Spans, func(i, j int) bool {
			return out.Spans[i].Seq < out.Spans[j].Seq
		})
	}
	return &out
}

func cloneSpan(in dto.BuildTraceSpan) dto.BuildTraceSpan {
	out := in
	if len(in.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(in.Attributes))
		for k, v := range in.Attributes {
			out.Attributes[k] = v
		}
	}
	return out
}

func attrsToMap(attrs []Attr) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		key := strings.TrimSpace(attr.Key)
		if key == "" {
			continue
		}
		out[key] = attr.Value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if len([]rune(text)) <= maxErrorLen {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxErrorLen]) + "...(truncated)"
}

func durationMillis(d time.Duration) int64 {
	ms := d.Milliseconds()
	if ms == 0 && d > 0 {
		return 1
	}
	return ms
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func newTraceID(now time.Time) string {
	var randomBytes [4]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Sprintf("%s-%d", now.Format("20060102T150405.000000000Z"), now.UnixNano())
	}
	return fmt.Sprintf("%s-%s", now.Format("20060102T150405.000000000Z"), hex.EncodeToString(randomBytes[:]))
}
