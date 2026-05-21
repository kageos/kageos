package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/discovery"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type appStartupPayload struct {
	Status       string    `json:"status"`
	StartTime    time.Time `json:"start_time"`
	ErrorMessage string    `json:"error_message"`
}

type appClosePayload struct {
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	CloseTime time.Time `json:"close_time"`
}

// AppDiscoveryHandler 只负责解码 runtime 生命周期事件并转交给 discovery service。
type AppDiscoveryHandler struct {
	service *AppDiscoveryService
}

func NewAppDiscoveryHandler(service *AppDiscoveryService) *AppDiscoveryHandler {
	return &AppDiscoveryHandler{service: service}
}

func (h *AppDiscoveryHandler) HandleRuntimeLifecycleEvent(msg *nats.Msg) {
	ctx := context.Background()

	var message subjects.Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[AppDiscoveryHandler] Failed to unmarshal runtime lifecycle event %s: %v", string(msg.Data), err)
		return
	}

	switch message.Type {
	case subjects.MessageTypeStatusStartup:
		payload, err := decodeLifecycleData[appStartupPayload](message.Data)
		if err != nil {
			logger.Errorf(ctx, "[AppDiscoveryHandler] Failed to decode startup payload: %v", err)
			return
		}
		h.service.applyStartupNotification(message.User, message.App, message.Version, payload.Status, payload.StartTime, payload.ErrorMessage)
	case subjects.MessageTypeStatusClose:
		payload, err := decodeLifecycleData[appClosePayload](message.Data)
		if err != nil {
			logger.Errorf(ctx, "[AppDiscoveryHandler] Failed to decode close payload: %v", err)
			return
		}
		h.service.applyCloseNotification(message.User, message.App, message.Version, payload.Status, payload.StartTime, payload.CloseTime)
	case subjects.MessageTypeStatusDiscovery:
		payload, err := decodeLifecycleData[discovery.DiscoveryResponse](message.Data)
		if err != nil {
			logger.Errorf(ctx, "[AppDiscoveryHandler] Failed to decode discovery payload: %v", err)
			return
		}
		payload.User = message.User
		payload.App = message.App
		payload.Version = message.Version
		h.service.applyDiscoveryResponse(&payload)
	default:
		logger.Warnf(ctx, "[AppDiscoveryHandler] Unknown message type: %s", message.Type)
	}
}

func decodeLifecycleData[T any](data interface{}) (T, error) {
	var target T

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return target, fmt.Errorf("marshal lifecycle data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &target); err != nil {
		return target, fmt.Errorf("unmarshal lifecycle data: %w", err)
	}

	return target, nil
}
