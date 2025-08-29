package application

import (
	"context"
	"strconv"
	"time"

	"github.com/davarch/ci-watcher/internal/domain"
)

type PollUseCase struct {
	gl    domain.GitlabClient
	note  domain.Notifier
	cache domain.StatusCache

	last map[domain.ProjectRef]struct {
		id     int64
		status domain.PipelineStatus
	}
}

func NewPollUseCase(gl domain.GitlabClient, note domain.Notifier, cache domain.StatusCache) *PollUseCase {
	return &PollUseCase{
		gl: gl, note: note, cache: cache,
		last: make(map[domain.ProjectRef]struct {
			id     int64
			status domain.PipelineStatus
		}),
	}
}

func (uc *PollUseCase) PollOnce(ctx context.Context, pr domain.ProjectRef) error {
	p, err := uc.gl.LatestPipeline(ctx, pr)
	if err != nil {
		return err
	}

	prev, ok := uc.last[pr]
	changed := !ok || prev.id != p.ID || prev.status != p.Status
	if changed {
		_ = uc.cache.Write(ctx, domain.Snapshot{
			Project: pr, Pipeline: p, Retrieved: time.Now().Unix(),
		})

		title := titleFor(p.Status)
		body := "Pipeline #" + strconv.FormatInt(p.ID, 10) + " (" + p.Ref + ")"
		_ = uc.note.Notify(ctx, title, body, p.WebURL)

		uc.last[pr] = struct {
			id     int64
			status domain.PipelineStatus
		}{p.ID, p.Status}
	}

	return nil
}

func titleFor(s domain.PipelineStatus) string {
	switch s {
	case domain.StatusSuccess:
		return "✅ CI: success"
	case domain.StatusFailed:
		return "❌ CI: failed"
	case domain.StatusRunning:
		return "▶️ CI: running"
	case domain.StatusCancelled:
		return "⛔ CI: canceled"
	default:
		return "ℹ️ CI: " + string(s)
	}
}
