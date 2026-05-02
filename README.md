# sd — smart directory

A smarter directory command for bash and zsh. Jump to directories by fuzzy name, bookmark, or history — without typing full paths.

## Features

- **Fuzzy jump** — type a partial directory name and jump instantly
- **Bookmarks** — save directories with a name and jump to them with `@name`
- **History** — frecency-ranked history with interactive selection
- **Stack** — pushd/popd style navigation
- **Zero hard dependencies** — pure Go binary, no `find`, `sqlite3`, or CGO required
- **fzf integration** — uses fzf when available, falls back to a built-in UI

## Installation

### Build from source

```bash
go install github.com/nshmdayo/sd/cmd/sd@latest
```

Or clone and build:

```bash
git clone https://github.com/nshmdayo/sd
cd sd
make build          # produces bin/sd
```

Move `bin/sd` somewhere on your `$PATH`, then add the shell integration to your RC file.

### Shell integration

**zsh** — add to `~/.zshrc`:

```zsh
eval "$(sd --init zsh)"
```

**bash** — add to `~/.bashrc`:

```bash
eval "$(sd --init bash)"
```

Restart your shell or `source ~/.zshrc` / `source ~/.bashrc`.

## Usage

### Fuzzy jump

```bash
cd proj          # jumps to the best-matching subdirectory named like "proj"
cd -g conf       # global search: searches from home directory
```

Candidate scoring is based on name similarity, directory depth, and visit frecency.
If multiple candidates match, an interactive selector (fzf or built-in) opens.

### Bookmarks

```bash
cd -a myproj     # bookmark current directory as "myproj"
cd @myproj       # jump to bookmark "myproj"
cd -l            # list all bookmarks
cd -d myproj     # delete bookmark "myproj"
cd -e            # edit bookmarks file in $EDITOR
```

Tab completion works for bookmark names:

```bash
cd @my<TAB>      # completes to @myproj
```

### History

```bash
cd -H            # interactive history browser (frecency order)
cd -1            # jump to most recent history entry
cd -3            # jump to third history entry
cd --clear-history  # delete all history
```

History is recorded automatically after every successful `cd`.

### Stack (pushd/popd)

```bash
cd -p /some/path # push path onto stack and jump to it
cd --            # pop: return to previous stack entry
cd -s            # show the current stack
```

### Other

```bash
cd               # go home (same as builtin cd)
cd --config      # edit config file in $EDITOR
cd --version     # print version
cd --help        # show help
```

## Configuration

Config file: `~/.config/smart-cd/config.toml` (created on first `cd --config`).

```toml
[search]
max_depth        = 5
global_root      = "~"
exclude_patterns = ["node_modules", ".git", "dist", ".cache"]

[history]
max_entries = 1000
sort        = "frecency"   # frecency | time | alpha

[ui]
color        = true
fuzzy_finder = "fzf"       # fzf | peco | internal
```

Environment variable overrides:

| Variable            | Effect                        |
|---------------------|-------------------------------|
| `SMART_CD_MAX_DEPTH`| Override `search.max_depth`   |
| `NO_COLOR`          | Disable color output          |

## Data files

| File                                         | Contents          |
|----------------------------------------------|-------------------|
| `~/.config/smart-cd/config.toml`             | Configuration     |
| `~/.config/smart-cd/bookmarks.json`          | Bookmarks         |
| `~/.local/share/smart-cd/history.db`         | Visit history (SQLite) |
| `~/.local/share/smart-cd/stack`              | Directory stack   |

XDG base directories (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`) are respected.

## How it works

`cd` is a shell builtin, so an external process cannot change the working directory directly.
`sd` handles all the logic and prints the resolved path to stdout. A thin shell wrapper captures it and calls `builtin cd`:

```
shell wrapper (cd function)
  └─ target=$(sd "$@" 2>/dev/tty)   ← captures path from stdout
     builtin cd "$target"
     sd --record "$target" &        ← record history asynchronously
```

UI output (fzf, error messages) goes to stderr so it reaches the terminal without polluting the captured path.

## Development

```bash
make test    # run all tests with race detector
make bench   # run benchmarks
make build   # build bin/sd
```

## Requirements

- Go 1.24+
- bash 4.0+ or zsh 5.0+
- macOS 12+ or Linux (Ubuntu 20.04+)
- [fzf](https://github.com/junegunn/fzf) (optional, recommended)

## License

MIT
