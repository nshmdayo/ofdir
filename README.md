# opfd — Operator of Files and DIRctories

A CLI tool for operating files and dirctories. Jump to directories by fuzzy name, bookmark, or history — without typing full paths.

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
go install github.com/nshmdayo/opfd/cmd/opfd@latest
```

Or clone and build:

```bash
git clone https://github.com/nshmdayo/opfd
cd opfd
make build          # produces bin/opfd
```

Move `bin/opfd` somewhere on your `$PATH`.

## Usage

### Fuzzy jump

```bash
opfd proj          # jump to best-matching directory (via shell function)
opfd -g conf       # global search: searches from home directory
```

Candidate scoring is based on name similarity, directory depth, and visit frecency.
If multiple candidates match, an interactive selector (fzf or built-in) opens.

### Bookmarks

```bash
opfd -a myproj     # bookmark current directory as "myproj"
opfd @myproj       # jump to bookmark "myproj"
opfd -l            # list all bookmarks
opfd -d myproj     # delete bookmark "myproj"
opfd -e            # edit bookmarks file in $EDITOR
```

Tab completion works for bookmark names:

```bash
opfd @myproj
```

### History

```bash
opfd -H            # interactive history browser (frecency order)
opfd -1            # jump to most recent history entry
opfd -3            # jump to third history entry
opfd --clear-history  # delete all history
```

History is recorded automatically after every successful `cd`.

### Stack (pushd/popd)

```bash
opfd -p /some/path # push path onto stack and jump to it
opfd --            # pop: return to previous stack entry
opfd -s            # show the current stack
```

### Other

```bash
opfd               # go home
opfd --config      # edit config file in $EDITOR
opfd --version     # print version
opfd --help        # show help
```

## Configuration

Config file: `~/.config/opfd/config.toml` (created on first `opfd --config`).

```bash
opfd --config      # create (if needed) and edit config file in $EDITOR
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
| `OFDIR_MAX_DEPTH`| Override `search.max_depth`   |
| `NO_COLOR`          | Disable color output          |

## Data files

| File                                         | Contents          |
|----------------------------------------------|-------------------|
| `~/.config/opfd/config.toml`             | Configuration     |
| `~/.config/opfd/bookmarks.json`          | Bookmarks         |
| `~/.local/share/opfd/history.db`         | Visit history (SQLite) |
| `~/.local/share/opfd/stack`              | Directory stack   |

XDG base directories (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`) are respected.

## How it works

`opfd` resolves a destination directory and prints it to stdout.

Use the provided shell initialization (`opfd --init bash` or `opfd --init zsh`) so the shell function can `cd` in the current shell.

## Development

Go supports cross-platform builds, so this project defaults to **non-containerized** local builds.

```bash
make test    # run all tests with race detector
make bench   # run benchmarks
make build   # build bin/opfd
make lint    # run golangci-lint static analysis
```

## Requirements

- Go 1.24+
- bash 4.0+ or zsh 5.0+
- macOS 12+ or Linux (Ubuntu 20.04+)
- [golangci-lint](https://golangci-lint.run/welcome/install/)
- [fzf](https://github.com/junegunn/fzf) (optional, recommended)

## License

MIT
