package domain

import (
	"context"
)

type MockGitLab struct {
	Pipeline Pipeline
	Err      error
	Called   int
}

func (m *MockGitLab) LatestPipeline(ctx context.Context, ref ProjectRef) (Pipeline, error) {
	m.Called++
	if m.Err != nil {
		return Pipeline{}, m.Err
	}
	return m.Pipeline, nil
}

type MockNotifier struct {
	Messages []string
	Err      error
}

func (n *MockNotifier) Notify(ctx context.Context, title, body, url string) error {
	n.Messages = append(n.Messages, title+"|"+body+"|"+url)
	return n.Err
}

type MockCache struct {
	Snapshots []Snapshot
	Err       error
}

func (c *MockCache) Write(ctx context.Context, s Snapshot) error {
	if c.Err != nil {
		return c.Err
	}
	c.Snapshots = append(c.Snapshots, s)
	return nil
}
