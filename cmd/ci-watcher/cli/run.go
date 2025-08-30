package cli

import (
	"context"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/davarch/ci-watcher/internal/application"
	"github.com/davarch/ci-watcher/internal/domain"
	"github.com/davarch/ci-watcher/internal/infrastructure/cache_fs"
	"github.com/davarch/ci-watcher/internal/infrastructure/config"
	"github.com/davarch/ci-watcher/internal/infrastructure/gitlab_http"
	"github.com/davarch/ci-watcher/internal/infrastructure/logging"
	"github.com/davarch/ci-watcher/internal/infrastructure/notify_libnotify"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run polling scheduler",
	Run: func(cmd *cobra.Command, args []string) {
		log := logging.New()
		defer func() { _ = log.Sync() }()

		cfg, err := config.Load(cfgPath)
		if err != nil {
			log.Fatal("config", zap.Error(err))
		}

		gl := gitlab_http.New(cfg.GitLab.BaseURL, cfg.GitLab.Token, cfg.GitLab.Timeout)
		note := notify_libnotify.NewSoft()
		cache := cache_fs.New(cfg.Cache.Path)

		uc := application.NewPollUseCase(gl, note, cache)

		var refs []domain.ProjectRef
		for _, p := range cfg.Poll.Projects {
			if p.Enabled {
				refs = append(refs, domain.ProjectRef{ProjectID: p.ProjectID, Ref: p.Ref})
			}
		}
		if len(refs) == 0 {
			log.Fatal("no enabled projects")
		}

		sched := application.NewScheduler(log, uc, refs, cfg.Poll.Interval, cfg.Poll.PauseFile)
		watchAndReload(cfgPath, cfg.Poll.Interval, log, sched)

		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		log.Info("start",
			zap.String("version", version),
			zap.Int("projects", len(refs)),
			zap.Duration("every", cfg.Poll.Interval),
			zap.String("cache", cfg.Cache.Path),
			zap.String("gitlab", cfg.GitLab.BaseURL),
			zap.String("pause_file", cfg.Poll.PauseFile),
		)
		sched.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func watchAndReload(cfgPath string, _ time.Duration, log *zap.Logger, sched *application.Scheduler) {
	if cfgPath == "" {
		return
	}

	dir := filepath.Dir(cfgPath)
	base := filepath.Base(cfgPath)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Warn("fsnotify init failed", zap.Error(err))
		return
	}

	go func() {
		defer func() { _ = w.Close() }()

		var (
			timer *time.Timer
			_     bool
			fire  = func() {
				_ = false
				cfg, err := config.Load(cfgPath)
				if err != nil {
					log.Warn("config reload failed", zap.Error(err))
					return
				}
				var refs []domain.ProjectRef
				for _, p := range cfg.Poll.Projects {
					if p.Enabled {
						refs = append(refs, domain.ProjectRef{ProjectID: p.ProjectID, Ref: p.Ref})
					}
				}
				if len(refs) == 0 {
					log.Warn("config reload: no enabled projects")
				}
				sched.UpdateRefs(refs)

				go sched.Run(context.WithValue(context.Background(), struct{}{}, nil))
			}
		)

		startTimer := func() {
			if timer == nil {
				timer = time.AfterFunc(300*time.Millisecond, fire)
			} else {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(300 * time.Millisecond)
			}
			_ = true
		}

		if err := w.Add(dir); err != nil {
			log.Warn("fsnotify add dir failed", zap.String("dir", dir), zap.Error(err))
			return
		}

		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}

				if filepath.Base(ev.Name) != base {
					continue
				}

				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					startTimer()
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Warn("fsnotify error", zap.Error(err))
			}
		}
	}()
}
