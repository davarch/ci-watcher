package application

import (
	"context"
	"sync"
	"time"

	"github.com/davarch/ci-watcher/internal/domain"
	"go.uber.org/zap"
)

type Scheduler struct {
	log   *zap.Logger
	use   *PollUseCase
	every time.Duration

	mu   sync.RWMutex
	refs []domain.ProjectRef
}

func NewScheduler(l *zap.Logger, u *PollUseCase, refs []domain.ProjectRef, every time.Duration) *Scheduler {
	return &Scheduler{log: l, use: u, refs: refs, every: every}
}

func (s *Scheduler) UpdateRefs(refs []domain.ProjectRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refs = refs
	s.log.Info("config reloaded", zap.Int("projects", len(refs)))
}

func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.every)
	defer t.Stop()

	s.runAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runAll(ctx)
		}
	}
}

func (s *Scheduler) runAll(ctx context.Context) {
	s.mu.RLock()
	refs := make([]domain.ProjectRef, len(s.refs))
	copy(refs, s.refs)
	s.mu.RUnlock()

	for _, pr := range refs {
		if err := s.use.PollOnce(ctx, pr); err != nil {
			s.log.Warn("poll failed",
				zap.Int64("project", pr.ProjectID),
				zap.String("ref", pr.Ref),
				zap.Error(err),
			)
		}
	}
}
