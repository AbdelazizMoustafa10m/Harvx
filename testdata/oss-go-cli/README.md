# gosync

A fast file synchronization CLI tool written in Go.

## Features

- Two-way file sync between local directories
- Watch mode for real-time synchronization
- Configurable ignore patterns
- Dry-run mode for previewing changes

## Installation

```bash
go install github.com/example/gosync@latest
```

## Usage

```bash
# Sync two directories
gosync sync ./source ./destination

# Watch mode
gosync sync --watch ./source ./destination

# Dry run
gosync sync --dry-run ./source ./destination
```

## Configuration

Create a `.gosync.toml` in your project root:

```toml
[sync]
ignore = ["*.tmp", ".git/", "node_modules/"]
max_file_size = "50MB"
```

## License

MIT License