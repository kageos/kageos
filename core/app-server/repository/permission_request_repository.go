package repository

import (
	"context"
	"errors"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

var ErrPermissionRequestStateChanged = errors.New("permission request state changed")

type PermissionRequestRepository struct {
	db *gorm.DB
}

func NewPermissionRequestRepository(db *gorm.DB) *PermissionRequestRepository {
	return &PermissionRequestRepository{db: db}
}

func (r *PermissionRequestRepository) Create(ctx context.Context, request *model.WorkspacePermissionRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *PermissionRequestRepository) FindPending(
	ctx context.Context,
	tenantUser, app, requester, resourcePath string,
) (*model.WorkspacePermissionRequest, error) {
	var request model.WorkspacePermissionRequest
	err := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ? AND requester = ? AND resource_path = ? AND status = ?",
			tenantUser, app, requester, resourcePath, model.PermissionRequestStatusPending).
		Order("id DESC").
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *PermissionRequestRepository) ListMine(
	ctx context.Context,
	tenantUser, app, requester, status string,
) ([]*model.WorkspacePermissionRequest, error) {
	var requests []*model.WorkspacePermissionRequest
	query := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ? AND requester = ?", tenantUser, app, requester)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at DESC, id DESC").Find(&requests).Error
	return requests, err
}

func (r *PermissionRequestRepository) GetByID(
	ctx context.Context,
	id int64,
) (*model.WorkspacePermissionRequest, error) {
	return r.GetByIDWithDB(ctx, r.db, id)
}

func (r *PermissionRequestRepository) ListWorkspace(
	ctx context.Context,
	tenantUser, app string,
	statuses []string,
) ([]*model.WorkspacePermissionRequest, error) {
	var requests []*model.WorkspacePermissionRequest
	query := r.db.WithContext(ctx).Where("tenant_user = ? AND app = ?", tenantUser, app)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.Order("created_at DESC, id DESC").Find(&requests).Error
	return requests, err
}

func (r *PermissionRequestRepository) Transaction(
	ctx context.Context,
	fn func(tx *gorm.DB) error,
) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *PermissionRequestRepository) GetByIDWithDB(
	ctx context.Context,
	db *gorm.DB,
	id int64,
) (*model.WorkspacePermissionRequest, error) {
	var request model.WorkspacePermissionRequest
	err := db.WithContext(ctx).Where("id = ?", id).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *PermissionRequestRepository) UpdateDecisionWithDB(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fromStatus string,
	updates map[string]any,
) error {
	result := db.WithContext(ctx).
		Model(&model.WorkspacePermissionRequest{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrPermissionRequestStateChanged
	}
	return nil
}
