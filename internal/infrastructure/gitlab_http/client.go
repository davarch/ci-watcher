package gitlab_http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/davarch/ci-watcher/internal/domain"
)

type Client struct {
	baseUrl string
	token   string
	hc      *http.Client
}

func New(baseUrl string, token string, timeout time.Duration) *Client {
	tr := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Client{
		baseUrl: trimSlash(baseUrl),
		token:   token,
		hc:      &http.Client{Transport: tr, Timeout: timeout},
	}
}

type pipelineDTO struct {
	ID     int64  `json:"id"`
	Ref    string `json:"ref"`
	Status string `json:"status"`
	WebURL string `json:"web_url"`
}

func (c *Client) LatestPipeline(ctx context.Context, pr domain.ProjectRef) (domain.Pipeline, error) {
	var out domain.Pipeline

	op := func() error {
		listURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines?ref=%s&per_page=1",
			c.baseUrl, pr.ProjectID, pr.Ref)

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
		req.Header.Set("PRIVATE-TOKEN", c.token)

		resp, err := c.hc.Do(req)
		if err != nil {
			return err
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusTooManyRequests {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if sec, _ := strconv.Atoi(ra); sec > 0 {
					select {
					case <-time.After(time.Duration(sec) * time.Second):
					case <-ctx.Done():
						return ctx.Err()
					}
					return fmt.Errorf("retry after due to 429")
				}
			}

			return fmt.Errorf("gitlab 429")
		}

		if resp.StatusCode >= 500 {
			return fmt.Errorf("gitlab %s", resp.Status)
		}

		if resp.StatusCode >= 300 {
			return backoff.Permanent(fmt.Errorf("gitlab %s", resp.Status))
		}

		var list []pipelineDTO
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			return err
		}

		if len(list) == 0 {
			out = domain.Pipeline{ID: 0, Ref: pr.Ref, Status: domain.StatusOther}
			return nil
		}

		p := list[0]

		detailURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines/%d", c.baseUrl, pr.ProjectID, p.ID)
		dreg, _ := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
		dreg.Header.Set("PRIVATE-TOKEN", c.token)

		dresp, derr := c.hc.Do(dreg)
		if derr == nil && dresp.StatusCode < 300 {
			defer func() { _ = dresp.Body.Close() }()
			var d pipelineDTO
			if json.NewDecoder(dresp.Body).Decode(&d) == nil && d.WebURL != "" {
				p.WebURL = d.WebURL
			}
		}

		out = domain.Pipeline{
			ID:     p.ID,
			Ref:    p.Ref,
			Status: mapStatus(p.Status),
			WebURL: p.WebURL,
		}

		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 300 * time.Millisecond
	bo.MaxInterval = 2 * time.Second
	bo.MaxElapsedTime = 5 * time.Second

	if err := backoff.Retry(op, backoff.WithContext(bo, ctx)); err != nil {
		return domain.Pipeline{}, err
	}
	return out, nil
}

func mapStatus(s string) domain.PipelineStatus {
	switch s {
	case "success":
		return domain.StatusSuccess
	case "failed":
		return domain.StatusFailed
	case "running":
		return domain.StatusRunning
	case "canceled":
		return domain.StatusCancelled
	default:
		return domain.StatusOther
	}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
