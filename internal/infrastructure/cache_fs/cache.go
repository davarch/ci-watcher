package cache_fs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/davarch/ci-watcher/internal/domain"
)

type FSCache struct {
	path string
}

func New(path string) *FSCache { return &FSCache{path: path} }

func (c *FSCache) Write(_ context.Context, s domain.Snapshot) error {
	if c.path == "" {
		return errors.New("cache path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	type out struct {
		ProjectID int64  `json:"project_id"`
		Ref       string `json:"ref"`
		Pipeline  int64  `json:"pipeline_id"`
		Status    string `json:"status"`
		URL       string `json:"url"`
		Retrieved int64  `json:"retrieved"`
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(out{
		ProjectID: s.Project.ProjectID,
		Ref:       s.Project.Ref,
		Pipeline:  s.Pipeline.ID,
		Status:    string(s.Pipeline.Status),
		URL:       s.Pipeline.WebURL,
		Retrieved: s.Retrieved,
	})
}
