package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromYAMLAndEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	cfgFile := filepath.Join(tmp, "config.yaml")

	yaml := `
gitlab:
  base_url: https://example.com
  token: token-yaml
  timeout: 5s

poll:
  interval: 10s
  projects:
    - project_id: 1
      ref: main
      enabled: true

cache:
  path: /tmp/cache.json
`
	if err := os.WriteFile(cfgFile, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GITLAB_TOKEN", "token-env")
	defer os.Unsetenv("GITLAB_TOKEN")

	c, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.GitLab.Token != "token-env" {
		t.Errorf("env override failed, got %s", c.GitLab.Token)
	}
	if len(c.Poll.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(c.Poll.Projects))
	}
}
