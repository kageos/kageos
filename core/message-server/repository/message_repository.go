package repository

import (
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

type InboxListFilter struct {
	Status          string
	ThreadKey       string
	SourcePath      string
	IncludeChildren bool
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}
