# ofdir — Operator of Files and DIRctories

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
go install github.com/nshmdayo/ofdir/cmd/ofdir@latest
```

Or clone and build:

```bash
git clone https://github.com/nshmdayo/ofdir
cd ofdir
make build          # produces bin/ofdir
```

Move `bin/ofdir` somewhere on your `$PATH`.

## Usage

### Fuzzy jump

```bash
ofdir proj          # jump to best-matching directory (via shell function)
ofdir -g conf       # global search: searches from home directory
```

Candidate scoring is based on name similarity, directory depth, and visit frecency.
If multiple candidates match, an interactive selector (fzf or built-in) opens.

### Bookmarks

```bash
ofdir -a myproj     # bookmark current directory as "myproj"
ofdir @myproj       # jump to bookmark "myproj"
ofdir -l            # list all bookmarks
ofdir -d myproj     # delete bookmark "myproj"
ofdir -e            # edit bookmarks file in $EDITOR
```

Tab completion works for bookmark names:

```bash
ofdir @myproj
```

### History

```bash
ofdir -H            # interactive history browser (frecency order)
ofdir -1            # jump to most recent history entry
ofdir -3            # jump to third history entry
ofdir --clear-history  # delete all history
```

History is recorded automatically after every successful `cd`.

### Stack (pushd/popd)

```bash
ofdir -p /some/path # push path onto stack and jump to it
ofdir --            # pop: return to previous stack entry
ofdir -s            # show the current stack
```

### Other

```bash
ofdir               # go home
ofdir --config      # edit config file in $EDITOR
ofdir --version     # print version
ofdir --help        # show help
```

## Configuration

Config file: `~/.config/ofdir/config.toml` (created on first `ofdir --config`).

```bash
ofdir --config      # create (if needed) and edit config file in $EDITOR
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
| `~/.config/ofdir/config.toml`             | Configuration     |
| `~/.config/ofdir/bookmarks.json`          | Bookmarks         |
| `~/.local/share/ofdir/history.db`         | Visit history (SQLite) |
| `~/.local/share/ofdir/stack`              | Directory stack   |

XDG base directories (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`) are respected.

## How it works

`ofdir` resolves a destination directory and prints it to stdout.

Use the provided shell initialization (`ofdir --init bash` or `ofdir --init zsh`) so the shell function can `cd` in the current shell.

## Development

Go supports cross-platform builds, so this project defaults to **non-containerized** local builds.

```bash
make test    # run all tests with race detector
make bench   # run benchmarks
make build   # build bin/ofdir
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
