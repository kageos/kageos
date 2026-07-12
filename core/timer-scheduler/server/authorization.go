package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

func bindCreateTaskToAuthenticatedUser(c *gin.Context, req *scheduledsdk.CreateTaskRequest) {
	if req == nil {
		return
	}
	user := currentTimerRequestUser(c)
	req.CreatedBy = user
	req.RequestUser = user
	req.RequestUserDept = strings.TrimSpace(c.GetHeader(contextx.DepartmentFullPathHeader))
	req.IdempotencyKey = scopedTimerIdempotencyKey(user, req.IdempotencyKey)
	req.Metadata = timerMetadataWithAuthenticatedIdentity(c, req.Metadata)
}

func bindUpdateTaskToAuthenticatedUser(c *gin.Context, req *scheduledsdk.UpdateTaskRequest) {
	if req == nil {
		return
	}
	user := currentTimerRequestUser(c)
	dept := strings.TrimSpace(c.GetHeader(contextx.DepartmentFullPathHeader))
	req.RequestUser = &user
	req.RequestUserDept = &dept
	if req.Metadata != nil {
		metadata := timerMetadataWithAuthenticatedIdentity(c, *req.Metadata)
		req.Metadata = &metadata
	}
}

func timerMetadataWithAuthenticatedIdentity(c *gin.Context, input map[string]string) map[string]string {
	metadata := make(map[string]string, len(input)+6)
	for key, value := range input {
		metadata[key] = value
	}

	email := ""
	if raw, exists := c.Get("email"); exists {
		email, _ = raw.(string)
	}
	setOrDeleteTimerMetadata(metadata, scheduledsdk.MetadataRequestEmail, strings.TrimSpace(email))
	setOrDeleteTimerMetadata(metadata, scheduledsdk.MetadataCompanyCode, strings.TrimSpace(c.GetHeader(contextx.CompanyCodeHeader)))
	setOrDeleteTimerMetadata(metadata, scheduledsdk.MetadataCompanyName, strings.TrimSpace(c.GetHeader(contextx.CompanyNameHeader)))
	setOrDeleteTimerMetadata(metadata, scheduledsdk.MetadataCompanyLogoURL, strings.TrimSpace(c.GetHeader(contextx.CompanyLogoURLHeader)))
	leader := ""
	if raw, exists := c.Get(timerVerifiedLeaderKey); exists {
		leader, _ = raw.(string)
	}
	setOrDeleteTimerMetadata(metadata, scheduledsdk.MetadataLeaderUsername, strings.TrimSpace(leader))
	return metadata
}

func setOrDeleteTimerMetadata(metadata map[string]string, key, value string) {
	if value == "" {
		delete(metadata, key)
		return
	}
	metadata[key] = value
}

func scopedTimerIdempotencyKey(username, original string) string {
	username = strings.TrimSpace(username)
	original = strings.TrimSpace(original)
	if original == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(username + "\x00" + original))
	return "u1-" + hex.EncodeToString(sum[:])
}

func requireOwnedTimerTask(c *gin.Context, service *timerservice.Service, taskID int64) (*scheduledsdk.Task, bool) {
	task, err := service.GetTask(requestContext(c), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeTimerTaskNotFound(c)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load timer task"})
		}
		return nil, false
	}
	owner := timerTaskOwner(task)
	if owner == "" || owner != currentTimerRequestUser(c) {
		writeTimerTaskNotFound(c)
		return nil, false
	}
	return task, true
}

func timerTaskOwner(task *scheduledsdk.Task) string {
	if task == nil {
		return ""
	}
	if owner := strings.TrimSpace(task.CreatedBy); owner != "" {
		return owner
	}
	return strings.TrimSpace(task.RequestUser)
}

func currentTimerRequestUser(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.GetHeader(contextx.RequestUserHeader))
}

func writeTimerTaskNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "timer task not found"})
}
