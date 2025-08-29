package notify_libnotify

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Notifier struct {
	soft bool
}

func New() *Notifier     { return &Notifier{soft: false} }
func NewSoft() *Notifier { return &Notifier{soft: true} }

type Options struct {
	Urgency string
	Expire  time.Duration
}

func (n *Notifier) Notify(ctx context.Context, title, body, url string) error {
	if strings.TrimSpace(url) != "" {
		if body == "" {
			body = url
		} else {
			body = body + "\n" + url
		}
	}

	args := []string{
		"--app-name=ci-watcher",
		title, body,
	}

	cmd := exec.CommandContext(ctx, "notify-send", args...)
	if err := cmd.Run(); err != nil {
		if n.soft {
			return nil
		}
		return err
	}
	return nil
}

func (n *Notifier) NotifyWith(ctx context.Context, title, body, url string, opt Options) error {
	if strings.TrimSpace(url) != "" {
		if body == "" {
			body = url
		} else {
			body = body + "\n" + url
		}
	}

	args := []string{"--app-name=ci-watcher"}
	if opt.Urgency != "" {
		args = append(args, "--urgency="+opt.Urgency)
	}
	if opt.Expire > 0 {
		ms := strconv.Itoa(int(opt.Expire / time.Millisecond))
		args = append(args, "--expire-time="+ms)
	}
	args = append(args, title, body)

	cmd := exec.CommandContext(ctx, "notify-send", args...)
	if err := cmd.Run(); err != nil {
		if n.soft {
			return nil
		}
		return err
	}

	return nil
}
