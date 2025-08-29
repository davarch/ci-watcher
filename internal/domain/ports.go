package domain

import "context"

type GitlabClient interface {
	LatestPipeline(ctx context.Context, ref ProjectRef) (Pipeline, error)
}

type Notifier interface {
	Notify(ctx context.Context, title, body, url string) error
}

type StatusCache interface {
	Write(ctx context.Context, s Snapshot) error
}
