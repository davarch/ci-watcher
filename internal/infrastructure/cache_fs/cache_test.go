package cache_fs

import (
	"context"
	"os"
	"testing"

	"github.com/davarch/ci-watcher/internal/domain"
)

func TestCache_WriteCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/snap.json"

	c := New(path)
	s := domain.Snapshot{
		Project:   domain.ProjectRef{ProjectID: 1, Ref: "main"},
		Pipeline:  domain.Pipeline{ID: 1, Ref: "main", Status: domain.StatusSuccess},
		Retrieved: 123,
	}
	if err := c.Write(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
