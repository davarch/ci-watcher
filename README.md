# CI Watcher

**CI Watcher** is a small Go daemon that polls GitLab pipelines and:
- shows native desktop notifications (via `notify-send`);
- writes the latest pipeline status into a JSON cache file (for [Waybar](https://github.com/Alexays/Waybar));
- provides a CLI to enable/disable projects, list them, etc.;
- supports config hot-reload while running.

This is especially useful for developers who want a tiny **desktop CI monitor** (e.g. in Sway/Wayland with Waybar).

---

## Features
- Poll one or multiple GitLab projects/branches.
- Show `success` / `failed` / `running` / `canceled` notifications.
- System tray/Waybar integration with colored status indicator.
- CLI to:
    - `run` the scheduler,
    - `list` configured projects,
    - `enable`/`disable` projects quickly,
    - `version`, `completion`.
- Hot-reload of `config.yaml` — no restart required.
- Works with `systemd --user` for background service.

---

## Requirements
- Linux (tested on Manjaro + Sway).
- Go ≥ 1.22 (for building).
- GitLab [Personal Access Token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) with `read_api`.
- Packages:
    - `libnotify` (for `notify-send`),
    - a Wayland notification daemon (e.g. [`mako`](https://github.com/emersion/mako)) if you use Sway,
    - `jq` (for Waybar helper script).

---

## Installation

### Build from source
```bash
git clone https://github.com/davarch/ci-watcher.git
cd ci-watcher
go build -o bin/ci-watcher ./cmd/ci-watcher
```

Place the binary in your `$PATH`:
```bash
install -Dm755 bin/ci-watcher ~/.local/bin/ci-watcher
```

---

## Configuration

Create your personal config:
```bash
mkdir -p ~/.config/ci-watcher
cp config.example.yaml ~/.config/ci-watcher/config.yaml
```

Edit `~/.config/ci-watcher/config.yaml`:
```yaml
gitlab:
  base_url: https://gitlab.com       # or https://git.<yourcompany>.com
  token: glpat_xxx                   # your GitLab Personal Access Token
  timeout: 10s

poll:
  interval: 20s
  projects:
    - name: core
      project_id: 111111
      ref: main
      enabled: true
    - name: report
      project_id: 222222
      ref: develop
      enabled: false

cache:
  path: ~/.cache/ci_status.json
```

---

## Usage

### CLI commands
```bash
ci-watcher run                # start scheduler (poll pipelines)
ci-watcher list               # list projects
ci-watcher enable <name>      # enable project by name
ci-watcher disable <name>     # disable project by name
ci-watcher version            # show version
ci-watcher completion bash    # generate shell completion
```

Examples:
```bash
ci-watcher list --enabled
ci-watcher enable core
ci-watcher disable report
```

### Background service with systemd

Create `~/.config/systemd/user/ci-watcher.service`:
```ini
[Unit]
Description=CI Watcher (GitLab polling -> notify + waybar cache)
After=network-online.target

[Service]
ExecStart=%h/.local/bin/ci-watcher run --config %h/.config/ci-watcher/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

Enable and start:
```bash
systemctl --user daemon-reload
systemctl --user enable --now ci-watcher.service
```

Logs:
```bash
journalctl --user -u ci-watcher.service -f
```

To stop temporarily:
```bash
systemctl --user stop ci-watcher.service
```

---

## Waybar Integration

### 1. Helper script
Create `~/.local/bin/ci-watcher-waybar`:
```bash
#!/usr/bin/env bash
f="$HOME/.cache/ci_status.json"
if jq -e . "$f" >/dev/null 2>&1; then
  status=$(jq -r '.status // "no-ci"' "$f")
  text=$(jq -r '(.status // "no-ci") + " #" + ((.pipeline_id // 0)|tostring)' "$f")
  url=$(jq -r '.url // ""' "$f")
else
  status="no-ci"; text="no-ci"; url=""
fi
printf '{"text":"%s","class":"%s","tooltip":"%s"}\n' "$text" "$status" "$url"
```
Make it executable:
```bash
chmod +x ~/.local/bin/ci-watcher-waybar
```

### 2. Waybar config (`~/.config/waybar/config.jsonc`)
```jsonc
{
  "modules-right": ["custom/ci"],

  "custom/ci": {
    "exec": "~/.local/bin/ci-watcher-waybar",
    "interval": 3,
    "return-type": "json",
    "on-click": "bash -lc 'u=$(jq -r .url ~/.cache/ci_status.json 2>/dev/null); [ -n "$u" ] && xdg-open "$u" || true'",
    "tooltip": true
  }
}
```

### 3. Waybar style (`~/.config/waybar/style.css`)
```css
#custom-ci { padding: 0 8px; border-radius: 6px; }

#custom-ci.success  { background: rgba(30,160,60,.25);  color: #9be18a; }
#custom-ci.failed   { background: rgba(200,50,50,.25);  color: #ff8c8c; }
#custom-ci.running  { background: rgba(180,140,20,.25); color: #ffd27a; }
#custom-ci.canceled { background: rgba(120,120,120,.25); color: #cfcfcf; }
#custom-ci.no-ci    { opacity: .6; }
```

Reload Waybar, and you’ll see `success #12345` (or other status) colored accordingly.  
Clicking the block opens the pipeline URL in your browser.

---

## Development

```bash
go test ./...
go build -o bin/ci-watcher ./cmd/ci-watcher
```

Build with version info:
```bash
go build -trimpath   -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)"   -o bin/ci-watcher ./cmd/ci-watcher
```

---

## License
MIT — feel free to use, fork, and adapt.