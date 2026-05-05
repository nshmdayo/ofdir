# sd — smart directory

A standalone smart directory CLI. Jump to directories by fuzzy name, bookmark, or history — without typing full paths.

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

Move `bin/sd` somewhere on your `$PATH`.

## Usage

### Fuzzy jump

```bash
sd proj          # jump to best-matching directory (via shell function)
sd -g conf       # global search: searches from home directory
```

Candidate scoring is based on name similarity, directory depth, and visit frecency.
If multiple candidates match, an interactive selector (fzf or built-in) opens.

### Bookmarks

```bash
sd -a myproj     # bookmark current directory as "myproj"
sd @myproj       # jump to bookmark "myproj"
sd -l            # list all bookmarks
sd -d myproj     # delete bookmark "myproj"
sd -e            # edit bookmarks file in $EDITOR
```

Tab completion works for bookmark names:

```bash
sd @myproj
```

### History

```bash
sd -H            # interactive history browser (frecency order)
sd -1            # jump to most recent history entry
sd -3            # jump to third history entry
sd --clear-history  # delete all history
```

History is recorded automatically after every successful `cd`.

### Stack (pushd/popd)

```bash
sd -p /some/path # push path onto stack and jump to it
sd --            # pop: return to previous stack entry
sd -s            # show the current stack
```

### Other

```bash
sd               # go home
sd --config      # edit config file in $EDITOR
sd --version     # print version
sd --help        # show help
```

## Configuration

Config file: `~/.config/smart-cd/config.toml` (created on first `sd --config`).

```bash
sd --config      # create (if needed) and edit config file in $EDITOR
```

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

`sd` resolves a destination directory and prints it to stdout.

Use the provided shell initialization (`sd --init bash` or `sd --init zsh`) so the shell function can `cd` in the current shell.

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
