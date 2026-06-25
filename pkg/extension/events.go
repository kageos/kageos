package extension

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
)

const globalEventKey = "*"

type ResourceRef struct {
	Type         string `json:"type,omitempty"`
	FullCodePath string `json:"full_code_path,omitempty"`
	Ref          string `json:"ref,omitempty"`
	Name         string `json:"name,omitempty"`
}

type Event struct {
	Name      string      `json:"name"`
	Source    string      `json:"source,omitempty"`
	Action    string      `json:"action,omitempty"`
	Actor     string      `json:"actor,omitempty"`
	Resource  ResourceRef `json:"resource,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Time      time.Time   `json:"time,omitempty"`
	Payload   any         `json:"payload,omitempty"`
}

type Listener func(context.Context, Event) error

var listenerRegistry = struct {
	sync.RWMutex
	listeners map[string][]Listener
}{
	listeners: make(map[string][]Listener),
}

func RegisterListener(eventName string, listener Listener) {
	if listener == nil {
		panic("extension listener is nil")
	}
	key := normalizeEventKey(eventName)
	listenerRegistry.Lock()
	defer listenerRegistry.Unlock()
	listenerRegistry.listeners[key] = append(listenerRegistry.listeners[key], listener)
}

func RegisteredListeners(eventName string) []Listener {
	keys := eventKeys(eventName)
	listenerRegistry.RLock()
	defer listenerRegistry.RUnlock()
	out := make([]Listener, 0)
	for _, key := range keys {
		out = append(out, listenerRegistry.listeners[key]...)
	}
	return out
}

func Emit(ctx context.Context, event Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	event.Name = normalizeEventKey(event.Name)
	listeners := RegisteredListeners(event.Name)
	if len(listeners) == 0 {
		return nil
	}
	event = fillEventFromContext(ctx, event)
	errs := make([]error, 0)
	for _, listener := range listeners {
		if err := listener(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func fillEventFromContext(ctx context.Context, event Event) Event {
	if event.Source == "" {
		event.Source = strings.TrimSpace(contextx.GetAuditClientSource(ctx))
	}
	if event.Actor == "" {
		event.Actor = strings.TrimSpace(contextx.GetRequestUser(ctx))
	}
	if event.RequestID == "" {
		event.RequestID = strings.TrimSpace(contextx.GetTraceId(ctx))
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	return event
}

func AsPayload[T any](event Event) (T, bool) {
	payload, ok := event.Payload.(T)
	return payload, ok
}

func normalizeEventKey(eventName string) string {
	eventName = strings.ToLower(strings.TrimSpace(eventName))
	if eventName == "" {
		return globalEventKey
	}
	return eventName
}

func eventKeys(eventName string) []string {
	key := normalizeEventKey(eventName)
	if key == globalEventKey {
		return []string{globalEventKey}
	}
	return []string{globalEventKey, key}
}
