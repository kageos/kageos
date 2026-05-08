package service

import (
	"container/heap"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
)

// DueQueue 按「下次执行时间」排序的队列，用于兜底到点执行（类似 Redis ZSET，score=next_run_at）
// 调度器按 score <= now 取出到点任务执行，避免固定 tick 间隔漏执行
type DueQueue interface {
	// Push 加入或更新任务的下次执行时间（同一 taskID 只保留一次，按 nextRunAt 排序）
	Push(taskID int64, nextRunAt time.Time)
	// PopDue 取出所有 nextRunAt <= now 的任务 ID 并从队列移除；执行后需根据新 next_run_at 再 Push
	PopDue(now time.Time) []int64
	// Remove 移除任务（如取消）
	Remove(taskID int64)
	// Sync 清空并按 DB 中待执行任务同步（启动时调用）
	Sync(tasks []*model.ScheduledTask)
}

// memDueQueue 内存实现：小顶堆按 next_run_at，PopDue(now) 取出所有已到点的
type memDueQueue struct {
	mu      sync.Mutex
	h       dueHeap
	removed map[int64]struct{} // 已取消等，Pop 时跳过
}

type dueItem struct {
	id   int64
	next time.Time
}

type dueHeap []dueItem

func (h dueHeap) Len() int           { return len(h) }
func (h dueHeap) Less(i, j int) bool { return h[i].next.Before(h[j].next) }
func (h dueHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *dueHeap) Push(x any)        { *h = append(*h, x.(dueItem)) }
func (h *dueHeap) Pop() any {
	old := *h
	n := len(old)
	*h = old[0 : n-1]
	return old[n-1]
}

func NewMemDueQueue() DueQueue {
	return &memDueQueue{
		h:       make(dueHeap, 0),
		removed: make(map[int64]struct{}),
	}
}

func (q *memDueQueue) Push(taskID int64, nextRunAt time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.removed, taskID)
	// 简单做法：直接加入堆，同 id 可能有多份，PopDue 时用 removed 去重即可；若已取消会先 Remove 再被 Pop 到则跳过
	heap.Push(&q.h, dueItem{id: taskID, next: nextRunAt})
}

func (q *memDueQueue) PopDue(now time.Time) []int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	seen := make(map[int64]struct{})
	var ids []int64
	for q.h.Len() > 0 && !q.h[0].next.After(now) {
		item := heap.Pop(&q.h).(dueItem)
		if _, removed := q.removed[item.id]; removed {
			continue
		}
		if _, ok := seen[item.id]; !ok {
			seen[item.id] = struct{}{}
			ids = append(ids, item.id)
		}
	}
	return ids
}

func (q *memDueQueue) Remove(taskID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.removed[taskID] = struct{}{}
}

func (q *memDueQueue) Sync(tasks []*model.ScheduledTask) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.h = q.h[:0]
	q.removed = make(map[int64]struct{})
	for _, t := range tasks {
		if t.Status != "pending" || t.NextRunAt == nil {
			continue
		}
		heap.Push(&q.h, dueItem{id: t.ID, next: *t.NextRunAt})
	}
}
