package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/ai-agent-os/hub/backend/model"
	"github.com/ai-agent-os/hub/backend/repository"
)

type PubKeyService struct {
	pubKeyRepo *repository.PubKeyRepository
}

func NewPubKeyService(pubKeyRepo *repository.PubKeyRepository) *PubKeyService {
	return &PubKeyService{pubKeyRepo: pubKeyRepo}
}

// Generate 为用户生成一个新的 pub key，返回完整 key（仅此一次可见）
func (s *PubKeyService) Generate(username, name string) (*model.PubKey, string, error) {
	rawKey, err := generateRandomKey()
	if err != nil {
		return nil, "", fmt.Errorf("生成密钥失败: %w", err)
	}

	fullKey := "pk_" + rawKey

	pubKey := &model.PubKey{
		Username:  username,
		Name:      name,
		Key:       fullKey,
		KeyPrefix: fullKey[:12],
	}

	if err := s.pubKeyRepo.Create(pubKey); err != nil {
		return nil, "", fmt.Errorf("保存密钥失败: %w", err)
	}

	return pubKey, fullKey, nil
}

// ListByUsername 列出用户的所有 pub key（不含完整 key）
func (s *PubKeyService) ListByUsername(username string) ([]model.PubKey, error) {
	return s.pubKeyRepo.ListByUsername(username)
}

// Delete 删除指定的 pub key（校验归属）
func (s *PubKeyService) Delete(id int64, username string) error {
	return s.pubKeyRepo.DeleteByIDAndUsername(id, username)
}

// ValidateKey 验证 pub key 是否有效，返回对应的用户名
func (s *PubKeyService) ValidateKey(key string) (string, error) {
	pubKey, err := s.pubKeyRepo.GetByKey(key)
	if err != nil {
		return "", fmt.Errorf("无效的 pub key")
	}

	go func() {
		_ = s.pubKeyRepo.UpdateLastUsedAt(pubKey.ID)
	}()

	return pubKey.Username, nil
}

func generateRandomKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
