package repository

import (
	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/gorm"
)

// WorkspaceHandoffPacketRepository stores structured workspace role handoff packets.
type WorkspaceHandoffPacketRepository struct {
	db *gorm.DB
}

func NewWorkspaceHandoffPacketRepository(db *gorm.DB) *WorkspaceHandoffPacketRepository {
	return &WorkspaceHandoffPacketRepository{db: db}
}

func (r *WorkspaceHandoffPacketRepository) Create(packet *model.WorkspaceHandoffPacket) error {
	return r.db.Create(packet).Error
}

func (r *WorkspaceHandoffPacketRepository) GetByTargetSessionID(targetSessionID string) (*model.WorkspaceHandoffPacket, error) {
	var packet model.WorkspaceHandoffPacket
	if err := r.db.Where("target_session_id = ?", targetSessionID).First(&packet).Error; err != nil {
		return nil, err
	}
	return &packet, nil
}
