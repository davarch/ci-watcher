package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Project struct {
	ProjectID int64  `yaml:"project_id"`
	Ref       string `yaml:"ref"`
	Enabled   bool   `yaml:"enabled"`
	Name      string `yaml:"name,omitempty"`
}

type Config struct {
	GitLab struct {
		BaseURL string        `yaml:"base_url"`
		Token   string        `yaml:"token"`
		Timeout time.Duration `yaml:"timeout"`
	} `yaml:"gitlab"`

	Poll struct {
		Interval  time.Duration `yaml:"interval"`
		Projects  []Project     `yaml:"projects"`
		PauseFile string        `yaml:"pause_file"`
	} `yaml:"poll"`

	Cache struct {
		Path string `yaml:"path"`
	} `yaml:"cache"`
}

func Load(path string) (Config, error) {
	var c Config

	c.GitLab.BaseURL = "https://gitlab.com"
	c.GitLab.Timeout = 10 * time.Second
	c.Poll.Interval = 20 * time.Second
	c.Cache.Path = expandHome("~/.cache/ci_status.json")

	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(b, &c)
		}
	}

	if v := os.Getenv("GITLAB_BASE_URL"); v != "" {
		c.GitLab.BaseURL = v
	}

	if v := os.Getenv("GITLAB_TOKEN"); v != "" {
		c.GitLab.Token = v
	}

	if v := os.Getenv("GITLAB_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.GitLab.Timeout = d
		}
	}

	if v := os.Getenv("INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Poll.Interval = d
		}
	}

	if v := os.Getenv("CACHE_PATH"); v != "" {
		c.Cache.Path = expandHome(v)
	}

	if s := os.Getenv("GITLAB_PROJECTS"); s != "" {
		var ps []Project
		for _, item := range strings.Split(s, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			parts := strings.SplitN(item, ":", 2)
			if len(parts) != 2 {
				continue
			}
			id, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				continue
			}
			ps = append(ps, Project{ProjectID: id, Ref: parts[1], Enabled: true})
		}
		if len(ps) > 0 {
			c.Poll.Projects = ps
		}
	} else if v := os.Getenv("GITLAB_PROJECT_ID"); v != "" {
		if pid, err := strconv.ParseInt(v, 10, 64); err == nil {
			ref := getenv("GITLAB_REF", "main")
			c.Poll.Projects = []Project{{ProjectID: pid, Ref: ref, Enabled: true}}
		}
	}

	c.Cache.Path = expandHome(c.Cache.Path)
	if c.GitLab.BaseURL == "" {
		c.GitLab.BaseURL = "https://gitlab.com"
	}

	if c.Poll.Interval <= 0 {
		c.Poll.Interval = 20 * time.Second
	}

	if c.GitLab.Timeout <= 0 {
		c.GitLab.Timeout = 10 * time.Second
	}

	if c.GitLab.Token == "" {
		return c, errors.New("GITLAB_TOKEN is required")
	}

	if len(c.Poll.Projects) == 0 {
		return c, errors.New("no projects configured (YAML or ENV)")
	}

	if c.Poll.PauseFile == "" {
		c.Poll.PauseFile = expandHome("~/.cache/ci_paused")
	}

	return c, nil
}

func Save(path string, c Config) error {
	if path == "" {
		return errors.New("empty config path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	lockFile := path + ".lock"
	lf, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lf.Close() }()

	if runtime.GOOS != "windows" {
		if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
			return err
		}
		defer func() { _ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) }()
	}

	b, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	if _, err := f.Write(b); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, _ := os.UserHomeDir(); h != "" {
			return h + p[1:]
		}
	}
	return p
}
