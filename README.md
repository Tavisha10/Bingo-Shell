# myshell

A fully featured, cross-platform command-line shell built from scratch in Go.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/license-MIT-green)

---

## Features

- **Smart prompt** — shows current directory, project type, git branch, modified files, and time
- **SQLite history** — searchable command history with timestamps, working directory, and project context
- **Project detection** — auto-detects Node, Go, Rust, Python, Deno, Java, Ruby, and PHP projects
- **Tool auto-runner** — suggests and runs project-relevant tools (`npm install`, `go test`, `cargo build`)
- **Environment manager** — scoped env vars, `.env` file loading, auto-load on `cd`
- **File explorer** — color-coded directory tree with file preview and search
- **Syntax highlighter** — highlights Go, Python, JavaScript, TypeScript, and Rust files
- **Session restore** — saves working directory and env vars, restores on next launch
- **Plugin hooks** — run custom scripts on `on_cd`, `on_cmd`, `on_enter`, `on_startup`, `on_exit`
- **TOML config** — persistent settings, aliases, and default env vars
- **Tab autocomplete** — built-in commands, subcommands, and dynamic file completions
- **Command tree** — visualize pipe/chain structure of any command before running it
- **Cross-platform** — works on macOS, Linux, and Windows (WSL)

---

## Demo

```
~/Documents/myproject [go] (main ~2) 14:32 ❯ tools
  go project tools

  !  go mod tidy        Tidy dependencies
     go build           Build the project
     go test            Run all tests
     go fmt             Format code

  run with: tools run <name>

~/Documents/myproject [go] (main ~2) 14:32 ❯ history search git
  01/15 09:22  ~/Documents/myproject    git add .
  01/15 09:23  ~/Documents/myproject    git commit -m "init"
  01/15 09:24  ~/Documents/myproject    git push

~/Documents/myproject [go] (main ~2) 14:32 ❯ tree ls | grep .go | wc -l

  Pipeline
  ├── ls
  ├── grep .go
  └── wc -l
```

---

## Installation

**Requirements:** Go 1.21+

```bash
# clone the repo
git clone https://github.com/Tavisha10/Bingo-Shell.git
cd myshell

# install
bash install.sh

# run
myshell
```

**Optional — set as your default shell:**
```bash
echo "$(which myshell)" | sudo tee -a /etc/shells
chsh -s "$(which myshell)"
```

**Install without sudo (to ~/bin):**
```bash
mkdir -p ~/bin
go build -o ~/bin/myshell ./...
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

---

## Built-in Commands

| Command | Description |
|---|---|
| `cd [dir]` | Change directory. Supports `~`, `-`, and `~/path` |
| `history` | Searchable SQLite history with `search`, `here`, `stats` |
| `explore [path]` | Color-coded file tree with preview and search |
| `tools` | List and run project-specific tools |
| `env` | Manage environment variables and `.env` files |
| `tree <cmd>` | Visualize a command's pipe/chain structure |
| `highlight <file>` | Syntax highlight a source file |
| `render` | Draw tables, boxes, progress bars, and diffs |
| `status` | Shell and git status panel |
| `config` | View and edit shell configuration |
| `session` | Save and restore shell sessions |
| `hooks` | Register scripts to run on shell events |
| `platform` | Platform info and dev tool detection |
| `clear` | Clear the terminal screen |
| `help [cmd]` | Show help for any built-in command |

---

## Configuration

Config is stored at `~/.config/myshell/config.toml` and created automatically on first run.

```toml
[general]
history_limit = 5000

[prompt]
show_git     = true
show_time    = true
show_project = true
show_exit_code = true

[explorer]
show_hidden = false
max_depth   = 3

[autorunner]
enabled = true

[aliases]
ll = "ls -la"
gs = "git status"

[env]
NODE_ENV = "development"
```

Edit with:
```bash
config edit       # opens in $EDITOR
config reload     # reload without restarting
```

---

## Plugin Hooks

Drop `.sh` scripts into `~/.config/myshell/hooks/` named after the event:

```
~/.config/myshell/hooks/
  on_cd.sh        # fires on every directory change
  on_enter.sh     # fires when entering a project root
  on_startup.sh   # fires once on shell start
  on_exit.sh      # fires on shell exit
  on_cmd.sh       # fires after every command
```

Available variables in hook scripts:

| Variable | Description |
|---|---|
| `$MYSHELL_DIR` | Current directory |
| `$MYSHELL_PROJECT` | Detected project type |
| `$MYSHELL_PROJECT_NAME` | Project name |
| `$MYSHELL_CMD` | Last command run (on_cmd only) |

Or register hooks dynamically:
```bash
hooks add on_cd logger "echo moved to $MYSHELL_DIR"
hooks remove logger
```

---

## Project Structure

```
myshell/
├── main.go        # REPL loop, readline, entry point
├── detector.go    # Project type detection
├── history.go     # SQLite history engine
├── env.go         # Environment variable manager
├── cmdtree.go     # Command tree parser and visualizer
├── autorun.go     # Tool auto-runner
├── platform.go    # Cross-platform layer
├── explorer.go    # File explorer
├── highlight.go   # Syntax highlighter
├── renderer.go    # Output renderer (tables, boxes, diffs)
├── statusbar.go   # Git-aware prompt and status bar
├── config.go      # TOML config system
├── session.go     # Session save and restore
├── hooks.go       # Plugin hook system
├── help.go        # Help system
├── complete.go    # Tab autocomplete
├── install.sh     # Installer script
└── test.sh        # Cross-platform test suite
```

---

## Running Tests

```bash
bash test.sh
```

---

## Tech Stack

- **Language** — Go
- **Line editing** — [chzyer/readline](https://github.com/chzyer/readline) — arrow keys, Ctrl+R search, tab complete
- **Database** — [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure Go SQLite, no CGo
- **Config** — hand-rolled TOML parser (no dependencies)
- **Highlighting** — custom token-based highlighter (no dependencies)

---

## Roadmap

- [ ] AI command layer (`*fix this`, `*explain output`)
- [ ] TUI panels with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [ ] Remote session sync
- [ ] Plugin marketplace

---

## License

MIT — feel free to use, modify, and distribute.
