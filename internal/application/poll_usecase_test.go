package application

import (
	"context"
	"testing"

	"github.com/davarch/ci-watcher/internal/domain"
)

func TestPollOnce_NewPipelineTriggersNotifyAndCache(t *testing.T) {
	gl := &domain.MockGitLab{Pipeline: domain.Pipeline{ID: 1, Ref: "main", Status: domain.StatusSuccess, WebURL: "u"}}
	note := &domain.MockNotifier{}
	cache := &domain.MockCache{}

	uc := NewPollUseCase(gl, note, cache)

	err := uc.PollOnce(context.Background(), domain.ProjectRef{ProjectID: 42, Ref: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(note.Messages) != 1 {
		t.Errorf("expected 1 notification, got %d", len(note.Messages))
	}
	if len(cache.Snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(cache.Snapshots))
	}
}

func TestPollOnce_SamePipelineDoesNothing(t *testing.T) {
	gl := &domain.MockGitLab{Pipeline: domain.Pipeline{ID: 1, Ref: "main", Status: domain.StatusSuccess}}
	note := &domain.MockNotifier{}
	cache := &domain.MockCache{}
	uc := NewPollUseCase(gl, note, cache)

	_ = uc.PollOnce(context.Background(), domain.ProjectRef{ProjectID: 42, Ref: "main"})
	_ = uc.PollOnce(context.Background(), domain.ProjectRef{ProjectID: 42, Ref: "main"})

	if len(note.Messages) != 1 {
		t.Errorf("expected 1 notification total, got %d", len(note.Messages))
	}
}
