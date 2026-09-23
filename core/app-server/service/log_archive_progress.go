package service

import (
	"context"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/logger"
	"time"
)

type archiveProgressKey struct{}

func (s *LogArchiveService) Progress(ctx context.Context, id int64) (*model.LogArchiveProgress, error) {
	return s.repo.Progress(ctx, id)
}
func (s *LogArchiveService) saveProgress(ctx context.Context) {
	p, _ := ctx.Value(archiveProgressKey{}).(*model.LogArchiveProgress)
	if p == nil || p.ExecutionID <= 0 {
		return
	}
	p.UpdatedAt = time.Now()
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.repo.SaveProgress(bounded, p); err != nil {
		logger.Warnf(ctx, "[LogArchive] save progress: %v", err)
	}
}
func (s *LogArchiveService) setPhase(ctx context.Context, phase string, batch *model.LogArchiveBatch) {
	p, _ := ctx.Value(archiveProgressKey{}).(*model.LogArchiveProgress)
	if p == nil {
		return
	}
	p.Phase, p.BatchID, p.BatchRecords = phase, batch.ID, batch.RecordCount
	s.saveProgress(ctx)
}
func (s *LogArchiveService) updateRunProgress(ctx context.Context, out archiveRunSummary) {
	p, _ := ctx.Value(archiveProgressKey{}).(*model.LogArchiveProgress)
	if p == nil {
		return
	}
	p.Records, p.Batches, p.FailedBatches = out.Records, out.Batches, out.FailedBatches
	p.Phase = "selecting"
	s.saveProgress(ctx)
}
